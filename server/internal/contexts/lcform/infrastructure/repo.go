// Package infrastructure is the PG persistence for the lcform module. All queries
// run through tenantdb.TenantDB, which sets search_path to the caller's tenant
// schema — so the same SQL is automatically isolated per tenant.
package infrastructure

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/lcform/domain"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type Repo struct{ db *tenantdb.TenantDB }

func NewRepo(db *tenantdb.TenantDB) *Repo { return &Repo{db: db} }

// ---- Form definitions ----

func (r *Repo) CreateDef(ctx context.Context, d *domain.FormDef) error {
	fields, err := json.Marshal(d.Fields)
	if err != nil {
		return err
	}
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO lcform_def (id, code, name, icon, fields, status, created_by, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7,$8,$9)`,
			d.ID, d.Code, d.Name, d.Icon, string(fields), d.Status, d.CreatedBy, d.CreatedAt, d.UpdatedAt)
		return err
	})
}

func (r *Repo) UpdateDef(ctx context.Context, d *domain.FormDef) error {
	fields, err := json.Marshal(d.Fields)
	if err != nil {
		return err
	}
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE lcform_def SET name=$1, icon=$2, fields=$3::jsonb, status=$4, updated_at=now()
			 WHERE id=$5`,
			d.Name, d.Icon, string(fields), d.Status, d.ID)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) DeleteDef(ctx context.Context, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `DELETE FROM lcform_def WHERE id=$1`, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) ListDefs(ctx context.Context, includeArchived bool) ([]*domain.FormDef, error) {
	where := ""
	if !includeArchived {
		where = "WHERE d.status = 'active'"
	}
	var out []*domain.FormDef
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT d.id, d.code, d.name, d.icon, d.fields, d.status, d.created_by, d.created_at, d.updated_at,
			        COALESCE(c.cnt, 0)
			 FROM lcform_def d
			 LEFT JOIN (SELECT form_id, count(*) AS cnt FROM lcform_entry GROUP BY form_id) c ON c.form_id = d.id
			 `+where+`
			 ORDER BY d.created_at DESC`)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.FormDef{}
		for rows.Next() {
			d, err := scanDef(rows)
			if err != nil {
				return err
			}
			out = append(out, d)
		}
		return rows.Err()
	})
	return out, err
}

func (r *Repo) GetDef(ctx context.Context, id kernel.ID) (*domain.FormDef, error) {
	var d *domain.FormDef
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`SELECT d.id, d.code, d.name, d.icon, d.fields, d.status, d.created_by, d.created_at, d.updated_at,
			        COALESCE((SELECT count(*) FROM lcform_entry e WHERE e.form_id = d.id), 0)
			 FROM lcform_def d WHERE d.id=$1`, id)
		got, err := scanDef(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		d = got
		return nil
	})
	return d, err
}

// CodeExists reports whether a definition with this code already exists (optionally
// excluding a given id, for update checks).
func (r *Repo) CodeExists(ctx context.Context, code string, excludeID kernel.ID) (bool, error) {
	var exists bool
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM lcform_def WHERE code=$1 AND ($2 = '' OR id <> $2::uuid))`,
			code, string(excludeID)).Scan(&exists)
	})
	return exists, err
}

func scanDef(row pgx.Row) (*domain.FormDef, error) {
	d := &domain.FormDef{}
	var rawFields []byte
	if err := row.Scan(&d.ID, &d.Code, &d.Name, &d.Icon, &rawFields, &d.Status,
		&d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &d.EntryCount); err != nil {
		return nil, err
	}
	d.Fields = []domain.Field{}
	if len(rawFields) > 0 {
		_ = json.Unmarshal(rawFields, &d.Fields)
	}
	if d.Fields == nil {
		d.Fields = []domain.Field{}
	}
	return d, nil
}

// ---- Entries ----

func (r *Repo) CreateEntry(ctx context.Context, e *domain.Entry) error {
	data, err := json.Marshal(e.Data)
	if err != nil {
		return err
	}
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO lcform_entry (id, form_id, data, submitted_by, created_at)
			 VALUES ($1,$2,$3::jsonb,$4,$5)`,
			e.ID, e.FormID, string(data), e.SubmittedBy, e.CreatedAt)
		return err
	})
}

// ListEntries returns a page of entries plus the total matching count. Search,
// when set, does a case-insensitive substring match against the JSONB text.
func (r *Repo) ListEntries(ctx context.Context, f domain.EntryFilter, limit, offset int) ([]*domain.Entry, int, error) {
	where := "WHERE form_id = $1"
	args := []any{f.FormID}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		where += " AND data::text ILIKE $2"
	}
	var out []*domain.Entry
	var total int
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM lcform_entry `+where, args...).Scan(&total); err != nil {
			return err
		}
		pagedArgs := append(append([]any{}, args...), limit, offset)
		rows, err := tx.Query(ctx,
			`SELECT id, form_id, data, submitted_by, created_at FROM lcform_entry `+where+
				` ORDER BY created_at DESC LIMIT $`+itoa(len(args)+1)+` OFFSET $`+itoa(len(args)+2),
			pagedArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Entry{}
		for rows.Next() {
			e, err := scanEntry(rows)
			if err != nil {
				return err
			}
			out = append(out, e)
		}
		return rows.Err()
	})
	return out, total, err
}

// AllEntries returns every entry for a form (no paging) — used by CSV export.
func (r *Repo) AllEntries(ctx context.Context, formID kernel.ID) ([]*domain.Entry, error) {
	var out []*domain.Entry
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, form_id, data, submitted_by, created_at FROM lcform_entry
			 WHERE form_id=$1 ORDER BY created_at DESC`, formID)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Entry{}
		for rows.Next() {
			e, err := scanEntry(rows)
			if err != nil {
				return err
			}
			out = append(out, e)
		}
		return rows.Err()
	})
	return out, err
}

func scanEntry(row pgx.Row) (*domain.Entry, error) {
	e := &domain.Entry{}
	var rawData []byte
	if err := row.Scan(&e.ID, &e.FormID, &rawData, &e.SubmittedBy, &e.CreatedAt); err != nil {
		return nil, err
	}
	e.Data = map[string]any{}
	if len(rawData) > 0 {
		_ = json.Unmarshal(rawData, &e.Data)
	}
	if e.Data == nil {
		e.Data = map[string]any{}
	}
	return e, nil
}

// itoa is a tiny local helper to build positional placeholders without strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
