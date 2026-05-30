// Package infrastructure is the PG persistence for the approval module. All
// queries run through tenantdb.TenantDB, which sets search_path to the caller's
// tenant schema (with public as a fallback) — so the same SQL is automatically
// isolated per tenant. Role / role_grant live in public and are queried with an
// explicit public. qualifier; member / department live in the tenant schema.
package infrastructure

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/approval/domain"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type Repo struct{ db *tenantdb.TenantDB }

func NewRepo(db *tenantdb.TenantDB) *Repo { return &Repo{db: db} }

// WithTx runs fn inside a tenant-scoped transaction. Used by the service to
// resolve a flow's assignees (member / role / dept lookups) under one snapshot.
func (r *Repo) WithTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error { return fn(ctx, tx) })
}

// MarkRead acknowledges a cc task (no flow effect).
func (r *Repo) MarkRead(ctx context.Context, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE approval_task SET status='read', acted_at=now() WHERE id=$1 AND status='pending'`, id)
		return err
	})
}

// ---- Forms ----

func (r *Repo) CreateForm(ctx context.Context, f *domain.Form) error {
	fields, err := json.Marshal(f.Fields)
	if err != nil {
		return err
	}
	flow, err := json.Marshal(f.Flow)
	if err != nil {
		return err
	}
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO approval_form (id, code, name, icon, description, fields, flow, status, created_by, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,$11)`,
			f.ID, f.Code, f.Name, f.Icon, f.Description, string(fields), string(flow),
			f.Status, f.CreatedBy, f.CreatedAt, f.UpdatedAt)
		return err
	})
}

func (r *Repo) UpdateForm(ctx context.Context, f *domain.Form) error {
	fields, err := json.Marshal(f.Fields)
	if err != nil {
		return err
	}
	flow, err := json.Marshal(f.Flow)
	if err != nil {
		return err
	}
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE approval_form SET name=$1, icon=$2, description=$3, fields=$4::jsonb,
			        flow=$5::jsonb, status=$6, updated_at=now()
			 WHERE id=$7`,
			f.Name, f.Icon, f.Description, string(fields), string(flow), f.Status, f.ID)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (r *Repo) DeleteForm(ctx context.Context, id kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `DELETE FROM approval_form WHERE id=$1`, id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

const formCols = `id, code, name, icon, description, fields, flow, status, created_by, created_at, updated_at`

func scanForm(row pgx.Row) (*domain.Form, error) {
	f := &domain.Form{}
	var rawFields, rawFlow []byte
	err := row.Scan(&f.ID, &f.Code, &f.Name, &f.Icon, &f.Description, &rawFields, &rawFlow,
		&f.Status, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	f.Fields = decodeFields(rawFields)
	f.Flow = decodeFlow(rawFlow)
	return f, nil
}

func (r *Repo) ListForms(ctx context.Context, includeDisabled bool) ([]*domain.Form, error) {
	where := ""
	if !includeDisabled {
		where = "WHERE status = 'active'"
	}
	var out []*domain.Form
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+formCols+` FROM approval_form `+where+` ORDER BY created_at DESC`)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Form{}
		for rows.Next() {
			f, err := scanForm(rows)
			if err != nil {
				return err
			}
			out = append(out, f)
		}
		return rows.Err()
	})
	return out, err
}

func (r *Repo) GetForm(ctx context.Context, id kernel.ID) (*domain.Form, error) {
	var f *domain.Form
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `SELECT `+formCols+` FROM approval_form WHERE id=$1`, id)
		got, err := scanForm(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		f = got
		return nil
	})
	return f, err
}

// ---- Instances + Tasks (submit) ----

// SubmitInstance inserts the instance and its first batch of tasks in one
// transaction so a submitted approval always has its initial node tasks.
func (r *Repo) SubmitInstance(ctx context.Context, ins *domain.Instance, tasks []*domain.Task) error {
	fields, err := json.Marshal(ins.Fields)
	if err != nil {
		return err
	}
	flow, err := json.Marshal(ins.Flow)
	if err != nil {
		return err
	}
	data, err := json.Marshal(ins.Data)
	if err != nil {
		return err
	}
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`INSERT INTO approval_instance (id, form_id, form_name, fields, data, initiator_id, status, current_node, flow, created_at, finished_at)
			 VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6,$7,$8,$9::jsonb,$10,$11)`,
			ins.ID, ins.FormID, ins.FormName, string(fields), string(data), ins.InitiatorID,
			ins.Status, ins.CurrentNode, string(flow), ins.CreatedAt, ins.FinishedAt); err != nil {
			return err
		}
		return insertTasks(ctx, tx, tasks)
	})
}

func insertTasks(ctx context.Context, tx pgx.Tx, tasks []*domain.Task) error {
	for _, t := range tasks {
		if _, err := tx.Exec(ctx,
			`INSERT INTO approval_task (id, instance_id, node_index, assignee_id, type, mode, status, comment, acted_at, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			t.ID, t.InstanceID, t.NodeIndex, t.AssigneeID, t.Type, t.Mode, t.Status, t.Comment, t.ActedAt, t.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

// ---- Instance queries ----

const instCols = `id, form_id, form_name, fields, data, initiator_id, status, current_node, flow, created_at, finished_at`

func scanInstance(row pgx.Row) (*domain.Instance, error) {
	ins := &domain.Instance{}
	var rawFields, rawData, rawFlow []byte
	err := row.Scan(&ins.ID, &ins.FormID, &ins.FormName, &rawFields, &rawData, &ins.InitiatorID,
		&ins.Status, &ins.CurrentNode, &rawFlow, &ins.CreatedAt, &ins.FinishedAt)
	if err != nil {
		return nil, err
	}
	ins.Fields = decodeFields(rawFields)
	ins.Flow = decodeFlow(rawFlow)
	ins.Data = map[string]any{}
	if len(rawData) > 0 {
		_ = json.Unmarshal(rawData, &ins.Data)
	}
	return ins, nil
}

// GetInstance loads the instance plus its full task timeline (ordered by node).
func (r *Repo) GetInstance(ctx context.Context, id kernel.ID) (*domain.Instance, error) {
	var ins *domain.Instance
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `SELECT `+instCols+` FROM approval_instance WHERE id=$1`, id)
		got, err := scanInstance(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		ins = got
		// Initiator name (tenant member).
		_ = tx.QueryRow(ctx, `SELECT display_name FROM member WHERE id=$1`, ins.InitiatorID).Scan(&ins.InitiatorName)
		// Timeline of tasks + assignee names.
		rows, err := tx.Query(ctx,
			`SELECT t.id, t.instance_id, t.node_index, t.assignee_id, t.type, t.mode, t.status,
			        t.comment, t.acted_at, t.created_at, COALESCE(m.display_name,'')
			 FROM approval_task t LEFT JOIN member m ON m.id = t.assignee_id
			 WHERE t.instance_id=$1
			 ORDER BY t.node_index, t.created_at`, id)
		if err != nil {
			return err
		}
		defer rows.Close()
		ins.Tasks = []*domain.Task{}
		for rows.Next() {
			t := &domain.Task{}
			if err := rows.Scan(&t.ID, &t.InstanceID, &t.NodeIndex, &t.AssigneeID, &t.Type, &t.Mode,
				&t.Status, &t.Comment, &t.ActedAt, &t.CreatedAt, &t.AssigneeName); err != nil {
				return err
			}
			ins.Tasks = append(ins.Tasks, t)
		}
		return rows.Err()
	})
	return ins, err
}

// ListInitiated returns instances started by the member.
func (r *Repo) ListInitiated(ctx context.Context, member kernel.ID) ([]*domain.Instance, error) {
	return r.queryInstances(ctx,
		`SELECT `+instCols+` FROM approval_instance WHERE initiator_id=$1 ORDER BY created_at DESC LIMIT 500`, member)
}

func (r *Repo) queryInstances(ctx context.Context, q string, args ...any) ([]*domain.Instance, error) {
	var out []*domain.Instance
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Instance{}
		for rows.Next() {
			ins, err := scanInstance(rows)
			if err != nil {
				return err
			}
			out = append(out, ins)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	// Decorate with initiator names in a second pass (small lists).
	return r.decorateInitiators(ctx, out)
}

func (r *Repo) decorateInitiators(ctx context.Context, list []*domain.Instance) ([]*domain.Instance, error) {
	if len(list) == 0 {
		return list, nil
	}
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		for _, ins := range list {
			_ = tx.QueryRow(ctx, `SELECT display_name FROM member WHERE id=$1`, ins.InitiatorID).Scan(&ins.InitiatorName)
		}
		return nil
	})
	return list, err
}

// ListTasksForMember returns the member's task-shaped inbox (todo / done / cc),
// joined to the owning instance for name + status display.
func (r *Repo) ListTasksForMember(ctx context.Context, member kernel.ID, statuses []string, types []string) ([]*domain.Task, error) {
	var out []*domain.Task
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT t.id, t.instance_id, t.node_index, t.assignee_id, t.type, t.mode, t.status,
			        t.comment, t.acted_at, t.created_at,
			        i.form_name, i.status
			 FROM approval_task t JOIN approval_instance i ON i.id = t.instance_id
			 WHERE t.assignee_id=$1 AND t.status = ANY($2) AND t.type = ANY($3)
			 ORDER BY t.created_at DESC LIMIT 500`,
			member, statuses, types)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []*domain.Task{}
		for rows.Next() {
			t := &domain.Task{}
			if err := rows.Scan(&t.ID, &t.InstanceID, &t.NodeIndex, &t.AssigneeID, &t.Type, &t.Mode,
				&t.Status, &t.Comment, &t.ActedAt, &t.CreatedAt, &t.FormName, &t.InstanceStatus); err != nil {
				return err
			}
			out = append(out, t)
		}
		return rows.Err()
	})
	return out, err
}

// GetTask loads a single task (no joins).
func (r *Repo) GetTask(ctx context.Context, id kernel.ID) (*domain.Task, error) {
	var t *domain.Task
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`SELECT id, instance_id, node_index, assignee_id, type, mode, status, comment, acted_at, created_at
			 FROM approval_task WHERE id=$1`, id)
		x := &domain.Task{}
		err := row.Scan(&x.ID, &x.InstanceID, &x.NodeIndex, &x.AssigneeID, &x.Type, &x.Mode,
			&x.Status, &x.Comment, &x.ActedAt, &x.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		t = x
		return nil
	})
	return t, err
}

// ---- Act: mutate a task + (optionally) advance/finish the instance atomically ----

// ActResult bundles the post-act outcome so the service can publish events.
type ActResult struct {
	InstanceFinished bool
	InstanceStatus   string
	AdvancedToNode   int
}

// Act performs the whole decision in one transaction:
//   - mark the acting task with status/comment/acted_at
//   - if reject: reject the instance + cancel all other pending tasks
//   - if approve: check the node's countersign mode; when the node is satisfied,
//     either generate the next node's tasks (and bump current_node) or finish.
//
// resolver maps a node's assignees to member ids, run INSIDE the tx so role/dept
// lookups share the snapshot. The flow + initiator come from the instance.
func (r *Repo) Act(ctx context.Context, taskID kernel.ID, approve bool, comment string,
	resolve func(ctx context.Context, tx pgx.Tx, initiator kernel.ID, node domain.FlowNode) ([]kernel.ID, error),
	newID func() kernel.ID) (*ActResult, error) {

	res := &ActResult{}
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		// Lock the task row.
		var t domain.Task
		err := tx.QueryRow(ctx,
			`SELECT id, instance_id, node_index, assignee_id, type, mode, status
			 FROM approval_task WHERE id=$1 FOR UPDATE`, taskID).
			Scan(&t.ID, &t.InstanceID, &t.NodeIndex, &t.AssigneeID, &t.Type, &t.Mode, &t.Status)
		if err != nil {
			return err // pgx.ErrNoRows bubbles up
		}

		// Load + lock the instance.
		row := tx.QueryRow(ctx, `SELECT `+instCols+` FROM approval_instance WHERE id=$1 FOR UPDATE`, t.InstanceID)
		ins, err := scanInstance(row)
		if err != nil {
			return err
		}

		newStatus := domain.StatusApproved
		if !approve {
			newStatus = domain.StatusRejected
		}
		if _, err := tx.Exec(ctx,
			`UPDATE approval_task SET status=$1, comment=$2, acted_at=now() WHERE id=$3`,
			newStatus, comment, taskID); err != nil {
			return err
		}

		// Reject short-circuits the whole instance.
		if !approve {
			if _, err := tx.Exec(ctx,
				`UPDATE approval_instance SET status='rejected', finished_at=now() WHERE id=$1`, ins.ID); err != nil {
				return err
			}
			// Cancel sibling pending approve tasks at the current node.
			if _, err := tx.Exec(ctx,
				`UPDATE approval_task SET status='canceled' WHERE instance_id=$1 AND status='pending' AND type='approve'`,
				ins.ID); err != nil {
				return err
			}
			res.InstanceFinished = true
			res.InstanceStatus = domain.StatusRejected
			return nil
		}

		// Approve: is the current node satisfied? For mode=and, every pending approve
		// task at this node must be gone. For mode=or, one approval is enough.
		nodeSatisfied := true
		if t.Mode == domain.ModeAnd {
			var pending int
			if err := tx.QueryRow(ctx,
				`SELECT count(*) FROM approval_task
				 WHERE instance_id=$1 AND node_index=$2 AND type='approve' AND status='pending'`,
				ins.ID, t.NodeIndex).Scan(&pending); err != nil {
				return err
			}
			nodeSatisfied = pending == 0
		} else {
			// or: cancel the remaining pending approve siblings at this node.
			if _, err := tx.Exec(ctx,
				`UPDATE approval_task SET status='canceled'
				 WHERE instance_id=$1 AND node_index=$2 AND type='approve' AND status='pending' AND id<>$3`,
				ins.ID, t.NodeIndex, taskID); err != nil {
				return err
			}
		}
		if !nodeSatisfied {
			return nil // wait for the other countersigners
		}

		// Advance to the next approve node (skipping nothing — cc nodes also create
		// tasks but never block; we still materialize them so they show in inboxes).
		next := t.NodeIndex + 1
		for next < len(ins.Flow) {
			node := ins.Flow[next]
			assignees, err := resolve(ctx, tx, ins.InitiatorID, node)
			if err != nil {
				return err
			}
			var tasks []*domain.Task
			status := domain.StatusPending
			for _, a := range assignees {
				tasks = append(tasks, &domain.Task{
					ID: newID(), InstanceID: ins.ID, NodeIndex: next, AssigneeID: a,
					Type: node.Type, Mode: nodeMode(node), Status: status, CreatedAt: ins.CreatedAt,
				})
			}
			if err := insertTasksNow(ctx, tx, tasks); err != nil {
				return err
			}
			if node.Type == domain.NodeApprove && len(tasks) > 0 {
				// Stop at the first approve node that actually has approvers.
				if _, err := tx.Exec(ctx,
					`UPDATE approval_instance SET current_node=$1 WHERE id=$2`, next, ins.ID); err != nil {
					return err
				}
				res.AdvancedToNode = next
				return nil
			}
			// cc node (or approve node with no resolvable assignees): keep going.
			next++
		}

		// No more approve nodes → fully approved.
		if _, err := tx.Exec(ctx,
			`UPDATE approval_instance SET status='approved', current_node=$1, finished_at=now() WHERE id=$2`,
			next, ins.ID); err != nil {
			return err
		}
		res.InstanceFinished = true
		res.InstanceStatus = domain.StatusApproved
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// insertTasksNow inserts tasks with created_at = now() (used when advancing).
func insertTasksNow(ctx context.Context, tx pgx.Tx, tasks []*domain.Task) error {
	for _, t := range tasks {
		if _, err := tx.Exec(ctx,
			`INSERT INTO approval_task (id, instance_id, node_index, assignee_id, type, mode, status, comment, acted_at, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,'',NULL,now())`,
			t.ID, t.InstanceID, t.NodeIndex, t.AssigneeID, t.Type, t.Mode, t.Status); err != nil {
			return err
		}
	}
	return nil
}

// CancelInstance lets the initiator withdraw a still-pending request.
func (r *Repo) CancelInstance(ctx context.Context, id, initiator kernel.ID) error {
	return r.db.Transaction(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx,
			`UPDATE approval_instance SET status='canceled', finished_at=now()
			 WHERE id=$1 AND initiator_id=$2 AND status='pending'`, id, initiator)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		_, err = tx.Exec(ctx,
			`UPDATE approval_task SET status='canceled' WHERE instance_id=$1 AND status='pending' AND type='approve'`, id)
		return err
	})
}

// ---- Assignee resolution helpers (run inside the caller's tx) ----

// ResolveAssignees maps a flow node to a de-duplicated list of member ids using
// the tenant member / department tables and the public.role_grant table.
//   - user:        the literal member id (validated to exist)
//   - role:        every member granted role_code in this tenant
//   - dept_leader: the leader (matched by display_name) of the initiator's department
func (r *Repo) ResolveAssignees(ctx context.Context, tx pgx.Tx, tenantID, initiator kernel.ID, node domain.FlowNode) ([]kernel.ID, error) {
	switch node.AssigneeType {
	case domain.AssigneeUser:
		if node.AssigneeID == "" {
			return nil, nil
		}
		var id kernel.ID
		err := tx.QueryRow(ctx, `SELECT id FROM member WHERE id=$1`, node.AssigneeID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return []kernel.ID{id}, nil

	case domain.AssigneeRole:
		if node.RoleCode == "" {
			return nil, nil
		}
		rows, err := tx.Query(ctx,
			`SELECT g.member_id FROM public.role_grant g
			 JOIN public.role r ON r.id = g.role_id
			 WHERE g.tenant_id=$1 AND r.code=$2`, tenantID, node.RoleCode)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return collectIDs(rows)

	case domain.AssigneeDeptLeader:
		// initiator → dept_id → department.leader (a display name) → member.id.
		var deptID *kernel.ID
		err := tx.QueryRow(ctx, `SELECT dept_id FROM member WHERE id=$1`, initiator).Scan(&deptID)
		if errors.Is(err, pgx.ErrNoRows) || deptID == nil {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		var leaderName *string
		if err := tx.QueryRow(ctx, `SELECT leader FROM department WHERE id=$1`, *deptID).Scan(&leaderName); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		if leaderName == nil || *leaderName == "" {
			return nil, nil
		}
		rows, err := tx.Query(ctx, `SELECT id FROM member WHERE display_name=$1 LIMIT 1`, *leaderName)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return collectIDs(rows)
	}
	return nil, nil
}

func collectIDs(rows pgx.Rows) ([]kernel.ID, error) {
	seen := map[kernel.ID]bool{}
	var out []kernel.ID
	for rows.Next() {
		var id kernel.ID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, rows.Err()
}

// ---- Members directory (for assignee pickers in the form/flow editors) ----

type MemberRef struct {
	ID         kernel.ID `json:"id"`
	Name       string    `json:"name"`
	Department string    `json:"department"`
}

func (r *Repo) ListMembers(ctx context.Context) ([]MemberRef, error) {
	var out []MemberRef
	err := r.db.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, display_name, COALESCE(department,'') FROM member
			 WHERE status='active' ORDER BY display_name LIMIT 500`)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = []MemberRef{}
		for rows.Next() {
			var m MemberRef
			if err := rows.Scan(&m.ID, &m.Name, &m.Department); err != nil {
				return err
			}
			out = append(out, m)
		}
		return rows.Err()
	})
	return out, err
}

// ---- decode helpers ----

func decodeFields(raw []byte) []domain.Field {
	out := []domain.Field{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = []domain.Field{}
	}
	return out
}

func decodeFlow(raw []byte) []domain.FlowNode {
	out := []domain.FlowNode{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = []domain.FlowNode{}
	}
	return out
}

func nodeMode(n domain.FlowNode) string {
	if n.Mode == domain.ModeAnd {
		return domain.ModeAnd
	}
	return domain.ModeOr
}
