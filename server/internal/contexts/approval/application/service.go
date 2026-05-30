// Package application is the use-case layer for the approval module: form CRUD,
// submitting an instance (snapshot + first-node tasks), acting on a task (advance
// or finish the flow), and the inbox lists (todo / done / initiated / cc).
package application

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/approval/domain"
	"github.com/leo/iop/server/internal/contexts/approval/infrastructure"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
)

type Service struct {
	repo  *infrastructure.Repo
	bus   eventbus.Bus
	clock kernel.Clock
}

func NewService(repo *infrastructure.Repo, bus eventbus.Bus, clk kernel.Clock) *Service {
	return &Service{repo: repo, bus: bus, clock: clk}
}

// ---- Forms ----

type SaveFormCmd struct {
	ID          kernel.ID // empty = create
	Code        string
	Name        string
	Icon        string
	Description string
	Fields      []domain.Field
	Flow        []domain.FlowNode
	Status      string
	Actor       kernel.ID
}

func (s *Service) ListForms(ctx context.Context, includeDisabled bool) ([]*domain.Form, error) {
	return s.repo.ListForms(ctx, includeDisabled)
}

func (s *Service) GetForm(ctx context.Context, id kernel.ID) (*domain.Form, error) {
	f, err := s.repo.GetForm(ctx, id)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "approval.db_error", "操作失败", err)
	}
	if f == nil {
		return nil, errors.New(errors.KindNotFound, "approval.form_not_found", "审批模板不存在")
	}
	return f, nil
}

func (s *Service) CreateForm(ctx context.Context, cmd SaveFormCmd) (*domain.Form, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, errors.New(errors.KindParam, "approval.form_name_required", "模板名称不能为空")
	}
	if err := validateFlow(cmd.Flow); err != nil {
		return nil, err
	}
	now := s.clock.Now()
	f := &domain.Form{
		ID: kernel.NewID(), Code: strings.TrimSpace(cmd.Code), Name: name,
		Icon: cmd.Icon, Description: cmd.Description,
		Fields: normFields(cmd.Fields), Flow: normFlow(cmd.Flow),
		Status: normFormStatus(cmd.Status), CreatedBy: cmd.Actor,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateForm(ctx, f); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "approval.create_form_failed", "创建模板失败", err)
	}
	_ = s.bus.Publish(ctx, "approval.form_created", map[string]any{"form_id": f.ID})
	return f, nil
}

func (s *Service) UpdateForm(ctx context.Context, cmd SaveFormCmd) (*domain.Form, error) {
	if cmd.ID == "" {
		return nil, errors.New(errors.KindParam, "approval.form_id_required", "模板 id 不能为空")
	}
	if strings.TrimSpace(cmd.Name) == "" {
		return nil, errors.New(errors.KindParam, "approval.form_name_required", "模板名称不能为空")
	}
	if err := validateFlow(cmd.Flow); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetForm(ctx, cmd.ID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "approval.db_error", "操作失败", err)
	}
	if existing == nil {
		return nil, errors.New(errors.KindNotFound, "approval.form_not_found", "审批模板不存在")
	}
	existing.Name = strings.TrimSpace(cmd.Name)
	existing.Icon = cmd.Icon
	existing.Description = cmd.Description
	existing.Fields = normFields(cmd.Fields)
	existing.Flow = normFlow(cmd.Flow)
	existing.Status = normFormStatus(cmd.Status)
	if err := s.repo.UpdateForm(ctx, existing); err != nil {
		return nil, notFoundOr(err, "approval.form_not_found", "审批模板不存在")
	}
	return existing, nil
}

func (s *Service) DeleteForm(ctx context.Context, id kernel.ID) error {
	return notFoundOr(s.repo.DeleteForm(ctx, id), "approval.form_not_found", "审批模板不存在")
}

// ---- Submit ----

type SubmitCmd struct {
	TenantID  kernel.ID
	FormID    kernel.ID
	Data      map[string]any
	Initiator kernel.ID
}

func (s *Service) Submit(ctx context.Context, cmd SubmitCmd) (*domain.Instance, error) {
	form, err := s.repo.GetForm(ctx, cmd.FormID)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "approval.db_error", "操作失败", err)
	}
	if form == nil || form.Status != domain.FormActive {
		return nil, errors.New(errors.KindNotFound, "approval.form_not_found", "审批模板不存在或已停用")
	}
	// Required-field validation against the template schema.
	for _, f := range form.Fields {
		if f.Required {
			v, ok := cmd.Data[f.Key]
			if !ok || isEmpty(v) {
				return nil, errors.New(errors.KindParam, "approval.field_required", "必填项「"+f.Label+"」不能为空")
			}
		}
	}

	now := s.clock.Now()
	ins := &domain.Instance{
		ID: kernel.NewID(), FormID: form.ID, FormName: form.Name,
		Fields: form.Fields, Flow: form.Flow, Data: cmd.Data,
		InitiatorID: cmd.Initiator, Status: domain.StatusPending,
		CurrentNode: 0, CreatedAt: now,
	}

	// Resolve tasks for the leading nodes: materialize cc nodes and the FIRST
	// approve node (with assignees). If there is no approve node at all, the
	// instance is auto-approved on submit.
	tasks, firstApprove, autoApprove, err := s.buildInitialTasks(ctx, cmd.TenantID, ins)
	if err != nil {
		return nil, err
	}
	if autoApprove {
		ins.Status = domain.StatusApproved
		ins.CurrentNode = len(ins.Flow)
		ins.FinishedAt = &now
	} else {
		ins.CurrentNode = firstApprove
	}

	if err := s.repo.SubmitInstance(ctx, ins, tasks); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "approval.submit_failed", "提交审批失败", err)
	}
	_ = s.bus.Publish(ctx, "approval.instance_submitted", map[string]any{"instance_id": ins.ID, "initiator": cmd.Initiator})
	return s.repo.GetInstance(ctx, ins.ID)
}

// buildInitialTasks walks the flow from node 0, creating cc tasks until the first
// approve node that has resolvable assignees; that approve node's tasks are also
// created. Returns the index of the active approve node, or autoApprove=true when
// no approve node yields any assignee. Assignee resolution runs inside a single tx
// so role/dept lookups share one snapshot.
func (s *Service) buildInitialTasks(ctx context.Context, tenantID kernel.ID, ins *domain.Instance) (tasks []*domain.Task, firstApprove int, autoApprove bool, err error) {
	now := s.clock.Now()
	autoApprove = true
	txErr := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for i := 0; i < len(ins.Flow); i++ {
			node := ins.Flow[i]
			assignees, e := s.repo.ResolveAssignees(ctx, tx, tenantID, ins.InitiatorID, node)
			if e != nil {
				return e
			}
			for _, a := range assignees {
				tasks = append(tasks, &domain.Task{
					ID: kernel.NewID(), InstanceID: ins.ID, NodeIndex: i, AssigneeID: a,
					Type: node.Type, Mode: nodeMode(node), Status: domain.StatusPending, CreatedAt: now,
				})
			}
			if node.Type == domain.NodeApprove && len(assignees) > 0 {
				firstApprove = i
				autoApprove = false
				return nil // stop at the first real approve node
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, 0, false, errors.Wrap(errors.KindDatabase, "approval.resolve_failed", "解析审批人失败", txErr)
	}
	return tasks, firstApprove, autoApprove, nil
}

// ---- Act ----

type ActCmd struct {
	TenantID kernel.ID
	TaskID   kernel.ID
	Member   kernel.ID
	Approve  bool
	Comment  string
}

func (s *Service) Act(ctx context.Context, cmd ActCmd) error {
	// Ownership + state guard.
	t, err := s.repo.GetTask(ctx, cmd.TaskID)
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "approval.db_error", "操作失败", err)
	}
	if t == nil {
		return errors.New(errors.KindNotFound, "approval.task_not_found", "审批任务不存在")
	}
	if t.AssigneeID != cmd.Member {
		return errors.New(errors.KindForbidden, "approval.not_assignee", "无权处理该审批任务")
	}
	if t.Type == domain.NodeCC {
		// CC tasks are acknowledge-only.
		if err := s.repo.MarkRead(ctx, cmd.TaskID); err != nil {
			return errors.Wrap(errors.KindDatabase, "approval.db_error", "操作失败", err)
		}
		return nil
	}
	if t.Status != domain.StatusPending {
		return errors.New(errors.KindConflict, "approval.task_done", "该任务已处理")
	}

	resolve := func(ctx context.Context, tx pgx.Tx, initiator kernel.ID, node domain.FlowNode) ([]kernel.ID, error) {
		return s.repo.ResolveAssignees(ctx, tx, cmd.TenantID, initiator, node)
	}
	res, err := s.repo.Act(ctx, cmd.TaskID, cmd.Approve, strings.TrimSpace(cmd.Comment), resolve, kernel.NewID)
	if err != nil {
		return notFoundOr(err, "approval.task_not_found", "审批任务不存在")
	}
	if res.InstanceFinished {
		_ = s.bus.Publish(ctx, "approval.instance_finished", map[string]any{
			"instance_id": t.InstanceID, "status": res.InstanceStatus,
		})
	}
	return nil
}

func (s *Service) CancelInstance(ctx context.Context, id, initiator kernel.ID) error {
	return notFoundOr(s.repo.CancelInstance(ctx, id, initiator), "approval.instance_not_cancelable", "审批不存在或无法撤回")
}

// ---- Lists ----

func (s *Service) Inbox(ctx context.Context, q domain.TaskQuery) (any, error) {
	switch q.Type {
	case "initiated":
		return s.repo.ListInitiated(ctx, q.Member)
	case "cc":
		return s.repo.ListTasksForMember(ctx, q.Member, []string{domain.StatusPending, domain.StatusRead}, []string{domain.NodeCC})
	case "done":
		return s.repo.ListTasksForMember(ctx, q.Member, []string{domain.StatusApproved, domain.StatusRejected}, []string{domain.NodeApprove})
	default: // "todo"
		return s.repo.ListTasksForMember(ctx, q.Member, []string{domain.StatusPending}, []string{domain.NodeApprove})
	}
}

// GetInstance loads an instance + its task timeline and annotates each task /
// the instance with viewer-relative flags (Mine / CanCancel) so the UI can show
// the right action buttons without learning the caller's member id from elsewhere.
func (s *Service) GetInstance(ctx context.Context, id, viewer kernel.ID) (*domain.Instance, error) {
	ins, err := s.repo.GetInstance(ctx, id)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "approval.db_error", "操作失败", err)
	}
	if ins == nil {
		return nil, errors.New(errors.KindNotFound, "approval.instance_not_found", "审批不存在")
	}
	ins.CanCancel = ins.InitiatorID == viewer && ins.Status == domain.StatusPending
	for _, t := range ins.Tasks {
		t.Mine = t.AssigneeID == viewer
	}
	return ins, nil
}

func (s *Service) ListMembers(ctx context.Context) ([]infrastructure.MemberRef, error) {
	return s.repo.ListMembers(ctx)
}

// ---- helpers ----

func validateFlow(flow []domain.FlowNode) error {
	for _, n := range flow {
		switch n.Type {
		case domain.NodeApprove, domain.NodeCC:
		default:
			return errors.New(errors.KindParam, "approval.bad_node_type", "审批节点类型无效")
		}
		switch n.AssigneeType {
		case domain.AssigneeUser, domain.AssigneeRole, domain.AssigneeDeptLeader:
		default:
			return errors.New(errors.KindParam, "approval.bad_assignee_type", "审批人类型无效")
		}
		if n.AssigneeType == domain.AssigneeUser && strings.TrimSpace(n.AssigneeID) == "" {
			return errors.New(errors.KindParam, "approval.assignee_required", "请选择审批人")
		}
		if n.AssigneeType == domain.AssigneeRole && strings.TrimSpace(n.RoleCode) == "" {
			return errors.New(errors.KindParam, "approval.role_required", "请选择审批角色")
		}
	}
	return nil
}

func normFields(in []domain.Field) []domain.Field {
	out := make([]domain.Field, 0, len(in))
	for _, f := range in {
		f.Key = strings.TrimSpace(f.Key)
		f.Label = strings.TrimSpace(f.Label)
		if f.Key == "" || f.Label == "" {
			continue
		}
		if f.Type == "" {
			f.Type = "text"
		}
		if f.Options == nil {
			f.Options = []string{}
		}
		out = append(out, f)
	}
	return out
}

func normFlow(in []domain.FlowNode) []domain.FlowNode {
	out := make([]domain.FlowNode, 0, len(in))
	for _, n := range in {
		if n.Type == "" {
			n.Type = domain.NodeApprove
		}
		n.Mode = nodeMode(n)
		out = append(out, n)
	}
	return out
}

func normFormStatus(s string) string {
	if s == domain.FormDisabled {
		return domain.FormDisabled
	}
	return domain.FormActive
}

func nodeMode(n domain.FlowNode) string {
	if n.Mode == domain.ModeAnd {
		return domain.ModeAnd
	}
	return domain.ModeOr
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(x) == ""
	case []any:
		return len(x) == 0
	}
	return false
}

func notFoundOr(err error, code, msg string) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, pgx.ErrNoRows) {
		return errors.New(errors.KindNotFound, code, msg)
	}
	return errors.Wrap(errors.KindDatabase, "approval.db_error", "操作失败", err)
}
