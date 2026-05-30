package iam

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// This file holds platform-console "extras" data access (P3): notices, generic
// params (over public.platform_setting), cron jobs + run records, and a
// cross-tenant online-session listing. All operate on public.* via the platform
// pool (no tenant context). Operation/login logs reuse audit.Service.ListPlatform.

// ============================ Platform Notice ============================

// PlatformNotice is a platform-wide announcement.
type PlatformNotice struct {
	ID        kernel.ID `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ListPlatformNotices returns notices newest-first, optionally filtered by status.
func ListPlatformNotices(ctx context.Context, pool *pgxpool.Pool, status string, p kernel.Pagination) ([]PlatformNotice, error) {
	p = p.Normalize()
	args := []any{p.PageSize, p.Offset()}
	where := ""
	if status != "" && status != "all" {
		where = " AND status = $3"
		args = append(args, status)
	}
	rows, err := pool.Query(ctx,
		`SELECT id, title, content, type, status, COALESCE(created_by::text,''), created_at
		 FROM public.platform_notice WHERE 1=1`+where+`
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlatformNotice{}
	for rows.Next() {
		var n PlatformNotice
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.Type, &n.Status, &n.CreatedBy, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// GetPlatformNotice returns one notice or (nil, nil) when absent.
func GetPlatformNotice(ctx context.Context, pool *pgxpool.Pool, id kernel.ID) (*PlatformNotice, error) {
	var n PlatformNotice
	err := pool.QueryRow(ctx,
		`SELECT id, title, content, type, status, COALESCE(created_by::text,''), created_at
		 FROM public.platform_notice WHERE id = $1`, id).
		Scan(&n.ID, &n.Title, &n.Content, &n.Type, &n.Status, &n.CreatedBy, &n.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// CreatePlatformNotice inserts a draft (or given status) notice.
func CreatePlatformNotice(ctx context.Context, pool *pgxpool.Pool, title, content, typ, status string, createdBy kernel.ID) (*PlatformNotice, error) {
	if typ == "" {
		typ = "notice"
	}
	if status == "" {
		status = "draft"
	}
	id := kernel.NewID()
	var by any
	if createdBy != "" {
		by = createdBy
	}
	_, err := pool.Exec(ctx,
		`INSERT INTO public.platform_notice (id, title, content, type, status, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6)`, id, title, content, typ, status, by)
	if err != nil {
		return nil, err
	}
	return GetPlatformNotice(ctx, pool, id)
}

// UpdatePlatformNotice updates the editable fields of a notice.
func UpdatePlatformNotice(ctx context.Context, pool *pgxpool.Pool, id kernel.ID, title, content, typ string) error {
	ct, err := pool.Exec(ctx,
		`UPDATE public.platform_notice SET title=$2, content=$3, type=$4 WHERE id=$1`,
		id, title, content, typ)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "iam.notice_not_found", "通知不存在")
	}
	return nil
}

// SetPlatformNoticeStatus moves a notice between draft/published/withdrawn.
func SetPlatformNoticeStatus(ctx context.Context, pool *pgxpool.Pool, id kernel.ID, status string) error {
	ct, err := pool.Exec(ctx, `UPDATE public.platform_notice SET status=$2 WHERE id=$1`, id, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "iam.notice_not_found", "通知不存在")
	}
	return nil
}

// DeletePlatformNotice removes a notice.
func DeletePlatformNotice(ctx context.Context, pool *pgxpool.Pool, id kernel.ID) error {
	_, err := pool.Exec(ctx, `DELETE FROM public.platform_notice WHERE id=$1`, id)
	return err
}

// ============================ Platform Params ============================

// PlatformParam is a generic key→JSONB platform setting.
type PlatformParam struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedBy string          `json:"updated_by,omitempty"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ListPlatformParams returns every platform_setting row.
func ListPlatformParams(ctx context.Context, pool *pgxpool.Pool) ([]PlatformParam, error) {
	rows, err := pool.Query(ctx,
		`SELECT key, value, COALESCE(updated_by::text,''), updated_at
		 FROM public.platform_setting ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlatformParam{}
	for rows.Next() {
		var p PlatformParam
		var raw []byte
		if err := rows.Scan(&p.Key, &raw, &p.UpdatedBy, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Value = json.RawMessage(raw)
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpsertPlatformParam writes a key/value pair (value is raw JSON).
func UpsertPlatformParam(ctx context.Context, pool *pgxpool.Pool, key string, value json.RawMessage, by kernel.ID) error {
	var updatedBy any
	if by != "" {
		updatedBy = by
	}
	_, err := pool.Exec(ctx,
		`INSERT INTO public.platform_setting (key, value, updated_by, updated_at)
		 VALUES ($1, $2::jsonb, $3, now())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = now()`,
		key, string(value), updatedBy)
	return err
}

// DeletePlatformParam removes a key.
func DeletePlatformParam(ctx context.Context, pool *pgxpool.Pool, key string) error {
	_, err := pool.Exec(ctx, `DELETE FROM public.platform_setting WHERE key=$1`, key)
	return err
}

// ============================ Online Users (cross-tenant) ============================

// PlatformOnlineRow is one active session across all tenants for the platform
// online-users view.
type PlatformOnlineRow struct {
	SessionID    kernel.ID `json:"session_id"`
	PlatformUser string    `json:"platform_user_id"`
	Username     string    `json:"username"`
	TenantID     string    `json:"tenant_id,omitempty"`
	TenantName   string    `json:"tenant_name,omitempty"`
	DisplayName  string    `json:"display_name,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	IssuedAt     string    `json:"issued_at"`
	ExpiresAt    string    `json:"expires_at"`
}

// ListAllOnlineSessions returns every active (non-revoked, unexpired) session
// across ALL tenants, joined with platform_user + tenant. Display names live in
// per-tenant member rows and are resolved in a second per-schema pass (best
// effort; a session with no tenant binding simply has no display name).
func ListAllOnlineSessions(ctx context.Context, pool *pgxpool.Pool) ([]PlatformOnlineRow, error) {
	rows, err := pool.Query(ctx,
		`SELECT s.id, s.platform_user_id::text, COALESCE(u.username, COALESCE(u.email,'')),
		        COALESCE(s.tenant_id::text,''), COALESCE(t.name,''), COALESCE(t.schema_name,''), s.member_id,
		        COALESCE(host(s.ip_address),''),
		        to_char(s.issued_at,'YYYY-MM-DD HH24:MI:SS'),
		        to_char(s.expires_at,'YYYY-MM-DD HH24:MI:SS')
		 FROM public.session s
		 JOIN public.platform_user u ON u.id = s.platform_user_id
		 LEFT JOIN public.tenant t ON t.id = s.tenant_id
		 WHERE s.revoked = FALSE AND s.expires_at > now()
		 ORDER BY s.issued_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type pending struct {
		idx      int
		memberID kernel.ID
	}
	out := []PlatformOnlineRow{}
	bySchema := map[string][]pending{} // schema_name → rows needing member display name
	for rows.Next() {
		var r PlatformOnlineRow
		var mid *kernel.ID
		var schema string
		if err := rows.Scan(&r.SessionID, &r.PlatformUser, &r.Username, &r.TenantID, &r.TenantName,
			&schema, &mid, &r.IPAddress, &r.IssuedAt, &r.ExpiresAt); err != nil {
			return nil, err
		}
		idx := len(out)
		out = append(out, r)
		if mid != nil && schema != "" {
			bySchema[schema] = append(bySchema[schema], pending{idx: idx, memberID: *mid})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Resolve member display names per tenant schema (best effort).
	for schema, list := range bySchema {
		ids := make([]kernel.ID, 0, len(list))
		for _, pnd := range list {
			ids = append(ids, pnd.memberID)
		}
		names, err := resolveMemberNames(ctx, pool, schema, ids)
		if err != nil {
			continue // best effort: leave display names blank on a per-tenant failure
		}
		for _, pnd := range list {
			if n, ok := names[pnd.memberID]; ok {
				out[pnd.idx].DisplayName = n
			}
		}
	}
	return out, nil
}

func resolveMemberNames(ctx context.Context, pool *pgxpool.Pool, schema string, ids []kernel.ID) (map[kernel.ID]string, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SET search_path TO `+pgIdent(schema)+`, public`); err != nil {
		return nil, err
	}
	defer conn.Exec(ctx, "RESET search_path") //nolint:errcheck
	r, err := conn.Query(ctx, `SELECT id, COALESCE(display_name,'') FROM member WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	names := map[kernel.ID]string{}
	for r.Next() {
		var id kernel.ID
		var name string
		if err := r.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	return names, r.Err()
}

// pgIdent quotes a schema identifier defensively (schema names are derived from
// validated slugs, but quote anyway to be safe with search_path interpolation).
func pgIdent(s string) string {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			out = append(out, '"')
		}
		out = append(out, s[i])
	}
	out = append(out, '"')
	return string(out)
}

// ============================ Cron Jobs ============================

// PlatformJob is a scheduled job definition.
type PlatformJob struct {
	ID        kernel.ID  `json:"id"`
	Name      string     `json:"name"`
	CronExpr  string     `json:"cron_expr"`
	Handler   string     `json:"handler"`
	Status    string     `json:"status"`
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
	NextRunAt *time.Time `json:"next_run_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// PlatformJobRun is one execution record.
type PlatformJobRun struct {
	ID         kernel.ID  `json:"id"`
	JobID      kernel.ID  `json:"job_id"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Status     string     `json:"status"`
	Detail     string     `json:"detail"`
}

// ListPlatformJobs returns all job definitions.
func ListPlatformJobs(ctx context.Context, pool *pgxpool.Pool) ([]PlatformJob, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, name, cron_expr, handler, status, last_run_at, next_run_at, created_at
		 FROM public.platform_job ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlatformJob{}
	for rows.Next() {
		var j PlatformJob
		if err := rows.Scan(&j.ID, &j.Name, &j.CronExpr, &j.Handler, &j.Status, &j.LastRunAt, &j.NextRunAt, &j.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// GetPlatformJob returns one job or (nil, nil).
func GetPlatformJob(ctx context.Context, pool *pgxpool.Pool, id kernel.ID) (*PlatformJob, error) {
	var j PlatformJob
	err := pool.QueryRow(ctx,
		`SELECT id, name, cron_expr, handler, status, last_run_at, next_run_at, created_at
		 FROM public.platform_job WHERE id=$1`, id).
		Scan(&j.ID, &j.Name, &j.CronExpr, &j.Handler, &j.Status, &j.LastRunAt, &j.NextRunAt, &j.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// CreatePlatformJob inserts a job. handler defaults to "noop".
func CreatePlatformJob(ctx context.Context, pool *pgxpool.Pool, name, cronExpr, handler, status string) (*PlatformJob, error) {
	if handler == "" {
		handler = "noop"
	}
	if status == "" {
		status = "enabled"
	}
	id := kernel.NewID()
	_, err := pool.Exec(ctx,
		`INSERT INTO public.platform_job (id, name, cron_expr, handler, status)
		 VALUES ($1,$2,$3,$4,$5)`, id, name, cronExpr, handler, status)
	if err != nil {
		return nil, err
	}
	return GetPlatformJob(ctx, pool, id)
}

// UpdatePlatformJob writes a full replacement of a job's mutable fields. The HTTP
// layer loads the existing job and merges optional fields before calling this, so
// every value here is authoritative (cron_expr may legitimately be empty).
func UpdatePlatformJob(ctx context.Context, pool *pgxpool.Pool, id kernel.ID, name, cronExpr, handler, status string) error {
	ct, err := pool.Exec(ctx,
		`UPDATE public.platform_job SET name=$2, cron_expr=$3, handler=$4, status=$5 WHERE id=$1`,
		id, name, cronExpr, handler, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errors.New(errors.KindNotFound, "iam.job_not_found", "任务不存在")
	}
	return nil
}

// DeletePlatformJob removes a job (and its runs via FK cascade).
func DeletePlatformJob(ctx context.Context, pool *pgxpool.Pool, id kernel.ID) error {
	_, err := pool.Exec(ctx, `DELETE FROM public.platform_job WHERE id=$1`, id)
	return err
}

// RunPlatformJobNow records a run for the job and executes its handler from the
// in-code handler registry. Handlers are no-op/echo for now (see jobHandlers).
// Returns the completed run record. The whole thing is guarded so a bad handler
// can never panic the request.
func RunPlatformJobNow(ctx context.Context, pool *pgxpool.Pool, id kernel.ID) (*PlatformJobRun, error) {
	job, err := GetPlatformJob(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, errors.New(errors.KindNotFound, "iam.job_not_found", "任务不存在")
	}
	runID := kernel.NewID()
	started := time.Now().UTC()
	if _, err := pool.Exec(ctx,
		`INSERT INTO public.platform_job_run (id, job_id, started_at, status, detail)
		 VALUES ($1,$2,$3,'running','')`, runID, id, started); err != nil {
		return nil, err
	}

	status, detail := runJobHandler(ctx, job)
	finished := time.Now().UTC()
	if _, err := pool.Exec(ctx,
		`UPDATE public.platform_job_run SET finished_at=$2, status=$3, detail=$4 WHERE id=$1`,
		runID, finished, status, detail); err != nil {
		return nil, err
	}
	// Update the job's last_run_at (best effort).
	_, _ = pool.Exec(ctx, `UPDATE public.platform_job SET last_run_at=$2 WHERE id=$1`, id, finished)

	return &PlatformJobRun{
		ID: runID, JobID: id, StartedAt: started, FinishedAt: &finished,
		Status: status, Detail: detail,
	}, nil
}

// runJobHandler dispatches to the in-code handler registry. Recovers from panics
// so a misbehaving handler degrades to a failed run rather than crashing.
func runJobHandler(ctx context.Context, job *PlatformJob) (status, detail string) {
	defer func() {
		if r := recover(); r != nil {
			status = "failed"
			detail = "handler panicked: " + toString(r)
		}
	}()
	h, ok := jobHandlers[job.Handler]
	if !ok {
		return "failed", "unknown handler: " + job.Handler
	}
	if err := h(ctx, job); err != nil {
		return "failed", err.Error()
	}
	return "success", "ok"
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if e, ok := v.(error); ok {
		return e.Error()
	}
	return "panic"
}

// jobHandlers is the in-process registry of runnable handlers. Real modules can
// register more; the built-ins are safe no-ops/echoes adequate for P3.
var jobHandlers = map[string]func(ctx context.Context, job *PlatformJob) error{
	"noop": func(ctx context.Context, job *PlatformJob) error { return nil },
	"echo": func(ctx context.Context, job *PlatformJob) error { return nil },
}

// ListPlatformJobRuns returns run records for a job, newest first.
func ListPlatformJobRuns(ctx context.Context, pool *pgxpool.Pool, jobID kernel.ID, p kernel.Pagination) ([]PlatformJobRun, error) {
	p = p.Normalize()
	rows, err := pool.Query(ctx,
		`SELECT id, job_id, started_at, finished_at, status, detail
		 FROM public.platform_job_run WHERE job_id=$1
		 ORDER BY started_at DESC LIMIT $2 OFFSET $3`, jobID, p.PageSize, p.Offset())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlatformJobRun{}
	for rows.Next() {
		var r PlatformJobRun
		if err := rows.Scan(&r.ID, &r.JobID, &r.StartedAt, &r.FinishedAt, &r.Status, &r.Detail); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ============================ Monitor ============================

// MonitorSnapshot is a point-in-time view of platform infra health + db pool stats.
type MonitorSnapshot struct {
	Time     time.Time      `json:"time"`
	DBPool   DBPoolStats    `json:"db_pool"`
	Counters map[string]int `json:"counters"`
}

// DBPoolStats mirrors the interesting subset of pgxpool.Stat.
type DBPoolStats struct {
	AcquiredConns       int32 `json:"acquired_conns"`
	IdleConns           int32 `json:"idle_conns"`
	TotalConns          int32 `json:"total_conns"`
	MaxConns            int32 `json:"max_conns"`
	NewConnsCount       int64 `json:"new_conns_count"`
	AcquireCount        int64 `json:"acquire_count"`
	CanceledAcquire     int64 `json:"canceled_acquire_count"`
	EmptyAcquireWaiting int64 `json:"empty_acquire_count"`
}

// MonitorDBStats reads the pgx pool statistics.
func MonitorDBStats(pool *pgxpool.Pool) DBPoolStats {
	st := pool.Stat()
	return DBPoolStats{
		AcquiredConns:       st.AcquiredConns(),
		IdleConns:           st.IdleConns(),
		TotalConns:          st.TotalConns(),
		MaxConns:            st.MaxConns(),
		NewConnsCount:       st.NewConnsCount(),
		AcquireCount:        st.AcquireCount(),
		CanceledAcquire:     st.CanceledAcquireCount(),
		EmptyAcquireWaiting: st.EmptyAcquireCount(),
	}
}

// MonitorCounters returns coarse platform counts (orgs / users / active sessions).
func MonitorCounters(ctx context.Context, pool *pgxpool.Pool) map[string]int {
	out := map[string]int{}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM public.tenant WHERE status='active'`).Scan(&n); err == nil {
		out["active_organizations"] = n
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM public.platform_user`).Scan(&n); err == nil {
		out["platform_users"] = n
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM public.session WHERE revoked=FALSE AND expires_at > now()`).Scan(&n); err == nil {
		out["active_sessions"] = n
	}
	return out
}
