// Package infrastructure is the PG persistence for the tasks module. All queries
// run through tenantdb.TenantDB, which sets search_path to the caller's tenant
// schema — so the same SQL is automatically isolated per tenant.
package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/tasks/domain"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type Repo struct{ db *tenantdb.TenantDB }

func NewRepo(db *tenantdb.TenantDB) *Repo { return &Repo{db: db} }

// ---- Lists ----

func (r *Repo) CreateList(ctx context.Context, l *domain.TaskList) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO task_list (id, owner, name, color, sort_order, archived, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			l.ID, l.Owner, l.Name, l.Color, l.SortOrder, l.Archived, l.CreatedAt, l.UpdatedAt)
		return err
	})
}

func (r *Repo) UpdateList(ctx context.Context, owner, id kernel.ID, name, color string, sortOrder int, archived bool) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE task_list SET name=$1, color=$2, sort_order=$3, archived=$4, updated_at=now()
			 WHERE id=$5 AND owner=$6 AND deleted_at IS NULL`,
			name, color, sortOrder, archived, id, owner)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) DeleteList(ctx context.Context, owner, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE task_list
			 SET deleted_at=now(), updated_at=now()
			 WHERE id=$1 AND owner=$2 AND deleted_at IS NULL`, id, owner)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		_, err = tx.Exec(ctx,
			`WITH RECURSIVE tree AS (
			   SELECT id FROM task WHERE list_id=$1 AND owner=$2 AND deleted_at IS NULL
			   UNION ALL
			   SELECT t.id FROM task t
			   JOIN tree p ON t.parent_id = p.id
			   WHERE t.owner=$2 AND t.deleted_at IS NULL
			 )
			 UPDATE task
			 SET deleted_at=now(), updated_at=now()
			 WHERE id IN (SELECT id FROM tree)`, id, owner)
		if err != nil {
			return err
		}
		return nil
	})
}

func (r *Repo) ListLists(ctx context.Context, owner kernel.ID) ([]*domain.TaskList, error) {
	var out []*domain.TaskList
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT l.id, l.owner, l.name, l.color, l.sort_order, l.archived, l.created_at, l.updated_at,
			        COALESCE(c.total,0), COALESCE(c.done,0)
			 FROM task_list l
			 LEFT JOIN (
			   SELECT list_id,
			          count(*) FILTER (WHERE parent_id IS NULL) AS total,
			          count(*) FILTER (WHERE parent_id IS NULL AND status='done') AS done
			   FROM task WHERE deleted_at IS NULL GROUP BY list_id
			 ) c ON c.list_id = l.id
			 WHERE l.owner=$1 AND l.deleted_at IS NULL
			 ORDER BY l.archived, l.sort_order, l.created_at`, owner)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.TaskList{}
		for rows.Next() {
			l := &domain.TaskList{}
			if err := rows.Scan(&l.ID, &l.Owner, &l.Name, &l.Color, &l.SortOrder, &l.Archived,
				&l.CreatedAt, &l.UpdatedAt, &l.TaskCount, &l.DoneCount); err != nil {
				return err
			}
			out = append(out, l)
		}
		return rows.Err()
	})
	return out, err
}

func (r *Repo) GetList(ctx context.Context, owner, id kernel.ID) (*domain.TaskList, error) {
	var l *domain.TaskList
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`SELECT id, owner, name, color, sort_order, archived, created_at, updated_at
			 FROM task_list WHERE id=$1 AND owner=$2 AND deleted_at IS NULL`, id, owner)
		var x domain.TaskList
		err := row.Scan(&x.ID, &x.Owner, &x.Name, &x.Color, &x.SortOrder, &x.Archived, &x.CreatedAt, &x.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		l = &x
		return nil
	})
	return l, err
}

// ---- Tasks ----

const taskCols = `id, owner, list_id, parent_id, title, note, priority, status, due_date, completed_at, tags, sort_order, created_at, updated_at`

func scanTask(row pgx.Row) (*domain.Task, error) {
	t := &domain.Task{}
	err := row.Scan(&t.ID, &t.Owner, &t.ListID, &t.ParentID, &t.Title, &t.Note, &t.Priority,
		&t.Status, &t.DueDate, &t.CompletedAt, &t.Tags, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if t.Tags == nil {
		t.Tags = []string{}
	}
	return t, nil
}

func (r *Repo) CreateTask(ctx context.Context, t *domain.Task) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO task (`+taskCols+`)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			t.ID, t.Owner, t.ListID, t.ParentID, t.Title, t.Note, t.Priority, t.Status,
			t.DueDate, t.CompletedAt, t.Tags, t.SortOrder, t.CreatedAt, t.UpdatedAt)
		return err
	})
}

func (r *Repo) GetTask(ctx context.Context, owner, id kernel.ID) (*domain.Task, error) {
	var t *domain.Task
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `SELECT `+taskCols+` FROM task WHERE id=$1 AND owner=$2 AND deleted_at IS NULL`, id, owner)
		got, err := scanTask(row)
		if err != nil {
			return err
		}
		t = got
		if t == nil {
			return nil
		}
		// Load subtasks (owner-scoped as defense-in-depth so a mis-parented row
		// from another member can never surface in this owner's task detail).
		rows, err := tx.Query(ctx, `SELECT `+taskCols+` FROM task WHERE parent_id=$1 AND owner=$2 AND deleted_at IS NULL ORDER BY sort_order, created_at`, id, owner)
		if err != nil {
			return err
		}
		defer rows.Close()
		t.Subtasks = []*domain.Task{}
		for rows.Next() {
			st, err := scanTask(rows)
			if err != nil {
				return err
			}
			t.Subtasks = append(t.Subtasks, st)
		}
		return rows.Err()
	})
	return t, err
}

// UpdateTask persists the EDITABLE fields only. It deliberately does NOT touch
// status/completed_at — those are owned by SetTaskStatus — so a concurrent edit
// and a concurrent complete/reopen operate on disjoint columns and cannot clobber
// each other (no status-resurrection lost-update).
func (r *Repo) UpdateTask(ctx context.Context, t *domain.Task) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE task SET list_id=$1, title=$2, note=$3, priority=$4,
			        due_date=$5, tags=$6, sort_order=$7, updated_at=now()
			 WHERE id=$8 AND owner=$9 AND deleted_at IS NULL`,
			t.ListID, t.Title, t.Note, t.Priority, t.DueDate,
			t.Tags, t.SortOrder, t.ID, t.Owner)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// SetTaskStatus does a targeted status/completed_at update in a single statement,
// independent of the editable-field overwrite in UpdateTask.
func (r *Repo) SetTaskStatus(ctx context.Context, owner, id kernel.ID, status string, completedAt *time.Time) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE task SET status=$1, completed_at=$2, updated_at=now()
			 WHERE id=$3 AND owner=$4 AND deleted_at IS NULL`,
			status, completedAt, id, owner)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) DeleteTask(ctx context.Context, owner, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`WITH RECURSIVE tree AS (
			   SELECT id FROM task WHERE id=$1 AND owner=$2 AND deleted_at IS NULL
			   UNION ALL
			   SELECT t.id FROM task t
			   JOIN tree p ON t.parent_id = p.id
			   WHERE t.owner=$2 AND t.deleted_at IS NULL
			 )
			 UPDATE task
			 SET deleted_at=now(), updated_at=now()
			 WHERE id IN (SELECT id FROM tree)`,
			id, owner)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// ListTasks runs the list/smart-view query. Top-level only (parent_id IS NULL)
// unless f.IncludeSubs. Ordering: incomplete first, then priority desc, then due, then sort.
func (r *Repo) ListTasks(ctx context.Context, f domain.Filter) ([]*domain.Task, error) {
	where := []string{"owner = $1", "deleted_at IS NULL"}
	args := []any{f.Owner}
	add := func(clause string, val any) {
		args = append(args, val)
		where = append(where, clause+" $"+itoa(len(args)))
	}
	if !f.IncludeSubs {
		where = append(where, "parent_id IS NULL")
	}
	if f.ListID != nil {
		add("list_id =", *f.ListID)
	}
	if f.Status != "" {
		add("status =", f.Status)
	}
	if f.Tag != "" {
		args = append(args, f.Tag)
		where = append(where, "$"+itoa(len(args))+" = ANY(tags)")
	}
	// View filters operate on due_date / status.
	switch f.View {
	case "today":
		where = append(where, "status='todo'", "due_date IS NOT NULL", "due_date < (date_trunc('day', now()) + interval '1 day')")
	case "next7":
		where = append(where, "status='todo'", "due_date IS NOT NULL", "due_date < (date_trunc('day', now()) + interval '8 day')")
	case "overdue":
		where = append(where, "status='todo'", "due_date IS NOT NULL", "due_date < date_trunc('day', now())")
	case "completed":
		where = append(where, "status='done'")
	}

	q := `SELECT ` + taskCols + ` FROM task WHERE ` + join(where, " AND ") +
		` ORDER BY (status='done'), priority DESC, due_date NULLS LAST, sort_order, created_at LIMIT 500`

	var out []*domain.Task
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Task{}
		for rows.Next() {
			t, err := scanTask(rows)
			if err != nil {
				return err
			}
			out = append(out, t)
		}
		return rows.Err()
	})
	return out, err
}

// CountByView returns counts for the smart-view badges in one round trip.
func (r *Repo) CountByView(ctx context.Context, owner kernel.ID) (map[string]int, error) {
	out := map[string]int{}
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT
			  count(*) FILTER (WHERE status='todo' AND parent_id IS NULL) AS all_todo,
			  count(*) FILTER (WHERE status='todo' AND parent_id IS NULL AND due_date IS NOT NULL AND due_date < (date_trunc('day', now()) + interval '1 day')) AS today,
			  count(*) FILTER (WHERE status='todo' AND parent_id IS NULL AND due_date IS NOT NULL AND due_date < (date_trunc('day', now()) + interval '8 day')) AS next7,
			  count(*) FILTER (WHERE status='todo' AND parent_id IS NULL AND due_date IS NOT NULL AND due_date < date_trunc('day', now())) AS overdue,
			  count(*) FILTER (WHERE status='done' AND parent_id IS NULL) AS completed
			FROM task WHERE owner=$1 AND deleted_at IS NULL`, owner)
		var allTodo, today, next7, overdue, completed int
		if err := row.Scan(&allTodo, &today, &next7, &overdue, &completed); err != nil {
			return err
		}
		out["all"] = allTodo
		out["today"] = today
		out["next7"] = next7
		out["overdue"] = overdue
		out["completed"] = completed
		return nil
	})
	return out, err
}

// small local helpers to avoid importing strconv/strings in hot paths
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

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	s := parts[0]
	for _, p := range parts[1:] {
		s += sep + p
	}
	return s
}
