package audit

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
	"go.uber.org/zap"
)

// Entry is one audit record (lives in tenant_<slug>.audit_log).
type Entry struct {
	ID         kernel.ID `json:"id"`
	OccurredAt time.Time `json:"occurred_at"`
	Actor      string    `json:"actor"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resource_id"`
	TraceID    string    `json:"trace_id"`
	Detail     []byte    `json:"detail"`
	TenantID   kernel.ID `json:"-"`
	SchemaName string    `json:"-"`
}

// Service buffers audit entries and writes asynchronously.
// Subscribe wires Service onto eventbus to auto-capture domain events.
type Service struct {
	pool    *pgxpool.Pool  // for platform-scoped writes to public.platform_audit_log
	tenants *tenantLookup  // helper to translate TenantID → SchemaName (filled at wire time)
	tenant  *tenantdb.TenantDB
	logger  *zap.Logger

	buf  chan Entry
	wg   sync.WaitGroup
	mu   sync.Mutex
	stop chan struct{}
	open bool
}

// tenantLookup gives Service access to GetTenant without importing tenancy package directly.
type tenantLookup struct {
	get func(ctx context.Context, id kernel.ID) (schemaName string, ok bool)
}

// NewService creates an Audit service with capacity-bounded buffer.
// pool is used for platform-scoped writes to public.platform_audit_log.
func NewService(pool *pgxpool.Pool, tenant *tenantdb.TenantDB, lookup func(ctx context.Context, id kernel.ID) (string, bool), logger *zap.Logger) *Service {
	s := &Service{
		pool:    pool,
		tenants: &tenantLookup{get: lookup},
		tenant:  tenant,
		logger:  logger,
		buf:     make(chan Entry, 1024),
		stop:    make(chan struct{}),
		open:    true,
	}
	s.wg.Add(1)
	go s.worker()
	return s
}

// Subscribe registers the auditor as a wildcard listener.  Since our v1 bus has
// no glob support we subscribe on a curated topic list.
func (s *Service) Subscribe(bus eventbus.Bus, topics []string) {
	for _, topic := range topics {
		topic := topic
		bus.Subscribe(topic, func(ctx context.Context, e eventbus.Event) error {
			detail, _ := json.Marshal(e.Data)
			s.RecordEvent(ctx, Entry{
				Actor:    e.Actor,
				Action:   e.Topic,
				TraceID:  e.TraceID,
				Detail:   detail,
				TenantID: e.TenantID,
			})
			return nil
		})
	}
}

// RecordEvent enqueues an entry. Returns immediately. If queue is full,
// fallback to synchronous write to avoid data loss.
func (s *Service) RecordEvent(ctx context.Context, e Entry) {
	if e.ID == "" {
		e.ID = kernel.NewID()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	if e.Actor == "" {
		e.Actor = "system"
	}
	if e.TenantID != "" {
		schema, ok := s.tenants.get(ctx, e.TenantID)
		if ok {
			e.SchemaName = schema
		}
	}
	s.mu.Lock()
	open := s.open
	s.mu.Unlock()
	if !open {
		return
	}
	select {
	case s.buf <- e:
	default:
		// Buffer full → sync write (never drop on the floor).
		s.write(context.Background(), e)
	}
}

func (s *Service) worker() {
	defer s.wg.Done()
	for {
		select {
		case e := <-s.buf:
			s.write(context.Background(), e)
		case <-s.stop:
			// drain
			for {
				select {
				case e := <-s.buf:
					s.write(context.Background(), e)
				default:
					return
				}
			}
		}
	}
}

func (s *Service) write(ctx context.Context, e Entry) {
	if e.SchemaName == "" {
		// No tenant scope → skip tenant-side write (M5+ could write to platform_audit_log).
		return
	}
	ctx = tenantdb.WithTenant(ctx, &tenantdb.TenantContext{
		ID: string(e.TenantID), Slug: "", SchemaName: e.SchemaName, Status: "active",
	})
	err := s.tenant.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO audit_log (id, occurred_at, actor, action, resource, resource_id, trace_id, detail)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)`,
			e.ID, e.OccurredAt, e.Actor, e.Action, e.Resource, e.ResourceID, e.TraceID, string(e.Detail))
		return err
	})
	if err != nil {
		s.logger.Warn("audit write failed", zap.Error(err), zap.String("topic", e.Action))
	}
}

// Close flushes the buffer and stops the worker.
func (s *Service) Close() error {
	s.mu.Lock()
	if !s.open {
		s.mu.Unlock()
		return nil
	}
	s.open = false
	s.mu.Unlock()
	close(s.stop)
	s.wg.Wait()
	return nil
}

// ListByTenant queries audit log for current tenant ctx.
func (s *Service) ListByTenant(ctx context.Context, p kernel.Pagination) ([]Entry, error) {
	return s.ListByTenantFiltered(ctx, p, ListFilter{})
}

// ListFilter narrows an audit_log query. All fields are optional.
//   - Actor: exact actor match.
//   - ActionPrefix: action starts with this string (e.g. "iam." for login logs).
//   - ActionLike: action ILIKE pattern (caller supplies % wildcards). Used to
//     match the family of login event topics.
//   - From / To: occurred_at window (RFC3339 / 'YYYY-MM-DD ...' strings as the
//     DB parses them; passed straight to a timestamptz comparison).
type ListFilter struct {
	Actor        string
	Action       string
	ActionPrefix string
	ActionLike   string
	From         string
	To           string
}

// ListByTenantFiltered queries the current tenant's audit_log with optional
// actor/action/time filters, newest first, paginated.
func (s *Service) ListByTenantFiltered(ctx context.Context, p kernel.Pagination, f ListFilter) ([]Entry, error) {
	p = p.Normalize()
	out := []Entry{}
	err := s.tenant.Transaction(ctx, func(tx pgx.Tx) error {
		args := []any{p.PageSize, p.Offset()}
		where := ""
		idx := 3
		if f.Actor != "" {
			where += " AND actor = $" + strconv.Itoa(idx)
			args = append(args, f.Actor)
			idx++
		}
		if f.Action != "" {
			where += " AND action = $" + strconv.Itoa(idx)
			args = append(args, f.Action)
			idx++
		}
		if f.ActionPrefix != "" {
			where += " AND action LIKE $" + strconv.Itoa(idx)
			args = append(args, f.ActionPrefix+"%")
			idx++
		}
		if f.ActionLike != "" {
			where += " AND action ILIKE $" + strconv.Itoa(idx)
			args = append(args, f.ActionLike)
			idx++
		}
		if f.From != "" {
			where += " AND occurred_at >= $" + strconv.Itoa(idx)
			args = append(args, f.From)
			idx++
		}
		if f.To != "" {
			where += " AND occurred_at <= $" + strconv.Itoa(idx)
			args = append(args, f.To)
			idx++
		}
		rows, err := tx.Query(ctx,
			`SELECT id, occurred_at, actor, action, COALESCE(resource,''), COALESCE(resource_id,''), COALESCE(trace_id,''), detail
			 FROM audit_log WHERE 1=1`+where+`
			 ORDER BY occurred_at DESC LIMIT $1 OFFSET $2`,
			args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e Entry
			if err := rows.Scan(&e.ID, &e.OccurredAt, &e.Actor, &e.Action, &e.Resource, &e.ResourceID, &e.TraceID, &e.Detail); err != nil {
				return err
			}
			out = append(out, e)
		}
		return rows.Err()
	})
	return out, err
}

// PlatformEntry is a platform-scoped audit record (stored in public.platform_audit_log).
type PlatformEntry struct {
	ID             kernel.ID `json:"id"`
	OccurredAt     time.Time `json:"occurred_at"`
	Actor          string    `json:"actor"`
	ActorRole      string    `json:"actor_role"`
	Action         string    `json:"action"`
	Resource       string    `json:"resource"`
	ResourceID     string    `json:"resource_id"`
	Reason         string    `json:"reason"`
	GovernanceMode string    `json:"governance_mode"`
	TraceID        string    `json:"trace_id"`
	Detail         []byte    `json:"detail"`
}

// RecordPlatform writes a platform-scoped audit entry to public.platform_audit_log.
// Synchronous + best-effort: logs and swallows errors so it never breaks a request.
func (s *Service) RecordPlatform(ctx context.Context, e PlatformEntry) {
	// Detach from request cancellation: this is often called after the HTTP response
	// is written, when the request context may already be cancelled. The write is a
	// compliance signal we don't want to lose to a closed connection.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if e.ID == "" {
		e.ID = kernel.NewID()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	if e.Actor == "" {
		e.Actor = "system"
	}
	var detail any
	if len(e.Detail) > 0 {
		detail = string(e.Detail)
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO public.platform_audit_log
		   (id, occurred_at, actor, actor_role, action, resource, resource_id, reason, governance_mode, trace_id, detail)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb)`,
		e.ID, e.OccurredAt, e.Actor, e.ActorRole, e.Action, e.Resource, e.ResourceID, e.Reason, e.GovernanceMode, e.TraceID, detail)
	if err != nil {
		s.logger.Warn("platform audit write failed", zap.Error(err), zap.String("action", e.Action))
	}
}

// ListPlatform returns platform-scoped audit entries, newest first.
func (s *Service) ListPlatform(ctx context.Context, p kernel.Pagination) ([]PlatformEntry, error) {
	p = p.Normalize()
	rows, err := s.pool.Query(ctx,
		`SELECT id, occurred_at, actor, COALESCE(actor_role,''), action, COALESCE(resource,''),
		        COALESCE(resource_id,''), COALESCE(reason,''), COALESCE(governance_mode,''),
		        COALESCE(trace_id,''), COALESCE(detail,'null'::jsonb)
		 FROM public.platform_audit_log ORDER BY occurred_at DESC LIMIT $1 OFFSET $2`,
		p.PageSize, p.Offset())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlatformEntry{}
	for rows.Next() {
		var e PlatformEntry
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.Actor, &e.ActorRole, &e.Action, &e.Resource,
			&e.ResourceID, &e.Reason, &e.GovernanceMode, &e.TraceID, &e.Detail); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
