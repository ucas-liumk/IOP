package notification

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
	"go.uber.org/zap"
)

type Notification struct {
	ID        kernel.ID  `json:"id"`
	Recipient kernel.ID  `json:"recipient"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body,omitempty"`
	Payload   any        `json:"payload,omitempty"`
	Read      bool       `json:"read"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}

type SendCmd struct {
	Recipient kernel.ID
	Type      string
	Title     string
	Body      string
	Payload   any
}

// Service is the public API of the notification service.
type Service struct {
	tenant *tenantdb.TenantDB
	logger *zap.Logger
	clock  kernel.Clock
}

func NewService(t *tenantdb.TenantDB, logger *zap.Logger, clk kernel.Clock) *Service {
	return &Service{tenant: t, logger: logger, clock: clk}
}

func (s *Service) Send(ctx context.Context, cmd SendCmd) (*Notification, error) {
	n := &Notification{
		ID:        kernel.NewID(),
		Recipient: cmd.Recipient,
		Type:      cmd.Type,
		Title:     cmd.Title,
		Body:      cmd.Body,
		Payload:   cmd.Payload,
		CreatedAt: s.clock.Now(),
	}
	payload, _ := json.Marshal(cmd.Payload)
	err := s.tenant.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO notification (id, recipient, type, title, body, payload, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)`,
			n.ID, n.Recipient, n.Type, n.Title, n.Body, string(payload), n.CreatedAt)
		return err
	})
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (s *Service) ListUnread(ctx context.Context, recipient kernel.ID, p kernel.Pagination) ([]*Notification, error) {
	p = p.Normalize()
	var out []*Notification
	err := s.tenant.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, recipient, type, title, COALESCE(body,''), payload, read, created_at, read_at
			 FROM notification WHERE recipient = $1 AND NOT read ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			recipient, p.PageSize, p.Offset())
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			n := &Notification{}
			var raw []byte
			if err := rows.Scan(&n.ID, &n.Recipient, &n.Type, &n.Title, &n.Body, &raw, &n.Read, &n.CreatedAt, &n.ReadAt); err != nil {
				return err
			}
			_ = json.Unmarshal(raw, &n.Payload)
			out = append(out, n)
		}
		return rows.Err()
	})
	return out, err
}

func (s *Service) MarkRead(ctx context.Context, id kernel.ID) error {
	now := s.clock.Now()
	return s.tenant.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE notification SET read = TRUE, read_at = $1 WHERE id = $2`, now, id)
		return err
	})
}

// Wire registers event-handlers for known business topics.
// New topics in M4 (OKR) will extend this list.
func (s *Service) Wire(bus eventbus.Bus) {
	// Example: a fresh tenant gets a welcome notification to the platform admin role.
	bus.Subscribe("tenancy.member_joined", func(ctx context.Context, e eventbus.Event) error {
		data, _ := e.Data.(map[string]any)
		if data == nil {
			return nil
		}
		mid, _ := data["member_id"].(string)
		if mid == "" {
			return nil
		}
		// Skip if no tenant context (cross-tenant event will set it via bus.Event.TenantID).
		ctx = tenantdb.WithTenant(ctx, &tenantdb.TenantContext{
			ID: string(e.TenantID), Slug: "", SchemaName: schemaForTenant(ctx),
			Status: "active",
		})
		_, _ = s.Send(ctx, SendCmd{
			Recipient: kernel.ID(mid),
			Type:      "system.welcome",
			Title:     "欢迎加入",
			Body:      "您已加入新租户。可在工作台开始填写日报 / 周报。",
		})
		return nil
	})
}

// schemaForTenant pulls schema_name from the existing TenantContext if set.
// Called inside Wire's handler where ctx already has TenantContext from the dispatch step.
func schemaForTenant(ctx context.Context) string {
	if tc, ok := tenantdb.FromContext(ctx); ok {
		return tc.SchemaName
	}
	return ""
}
