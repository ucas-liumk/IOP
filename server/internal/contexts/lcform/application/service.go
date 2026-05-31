// Package application is the use-case layer for the lcform module: form-definition
// CRUD, entry submission (with schema-driven validation), paged entry listing and
// CSV export.
package application

import (
	"context"
	stderrors "errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/contexts/lcform/domain"
	"github.com/leo/iop/server/internal/contexts/lcform/infrastructure"
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

var codeRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)

// ---- Form definitions ----

type SaveDefCmd struct {
	Code      string // create-only; ignored on update
	Name      string
	Icon      string
	Fields    []domain.Field
	Status    string
	CreatedBy kernel.ID
}

func (s *Service) CreateDef(ctx context.Context, cmd SaveDefCmd) (*domain.FormDef, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, errors.New(errors.KindParam, "lcform.name_required", "表单名称不能为空")
	}
	code := strings.TrimSpace(strings.ToLower(cmd.Code))
	if code == "" {
		code = autoCode(name)
	}
	if !codeRe.MatchString(code) {
		return nil, errors.New(errors.KindParam, "lcform.code_invalid", "表单编码格式无效（小写字母/数字/-/_，最长 40）")
	}
	fields, err := normFields(cmd.Fields)
	if err != nil {
		return nil, err
	}
	if exists, err := s.repo.CodeExists(ctx, code, ""); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "lcform.db_error", "操作失败", err)
	} else if exists {
		return nil, errors.New(errors.KindConflict, "lcform.code_taken", "表单编码已被占用")
	}
	now := s.clock.Now()
	d := &domain.FormDef{
		ID: kernel.NewID(), Code: code, Name: name, Icon: strings.TrimSpace(cmd.Icon),
		Fields: fields, Status: normStatus(cmd.Status), CreatedBy: cmd.CreatedBy,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateDef(ctx, d); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "lcform.create_def_failed", "创建表单失败", err)
	}
	_ = s.bus.Publish(ctx, "lcform.form_created", map[string]any{"form_id": d.ID, "by": cmd.CreatedBy})
	return d, nil
}

func (s *Service) UpdateDef(ctx context.Context, id kernel.ID, cmd SaveDefCmd) (*domain.FormDef, error) {
	d, err := s.repo.GetDef(ctx, id)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "lcform.db_error", "操作失败", err)
	}
	if d == nil {
		return nil, errors.New(errors.KindNotFound, "lcform.form_not_found", "表单不存在")
	}
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, errors.New(errors.KindParam, "lcform.name_required", "表单名称不能为空")
	}
	fields, err := normFields(cmd.Fields)
	if err != nil {
		return nil, err
	}
	d.Name = name
	d.Icon = strings.TrimSpace(cmd.Icon)
	d.Fields = fields
	d.Status = normStatus(cmd.Status)
	if err := s.repo.UpdateDef(ctx, d); err != nil {
		return nil, notFoundOr(err, "lcform.form_not_found", "表单不存在")
	}
	return d, nil
}

func (s *Service) DeleteDef(ctx context.Context, id kernel.ID) error {
	return notFoundOr(s.repo.DeleteDef(ctx, id), "lcform.form_not_found", "表单不存在")
}

func (s *Service) ListDefs(ctx context.Context, includeArchived bool) ([]*domain.FormDef, error) {
	return s.repo.ListDefs(ctx, includeArchived)
}

func (s *Service) GetDef(ctx context.Context, id kernel.ID) (*domain.FormDef, error) {
	d, err := s.repo.GetDef(ctx, id)
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "lcform.db_error", "操作失败", err)
	}
	if d == nil {
		return nil, errors.New(errors.KindNotFound, "lcform.form_not_found", "表单不存在")
	}
	return d, nil
}

// ---- Entries ----

// SubmitEntry validates the payload against the form schema and persists one record.
func (s *Service) SubmitEntry(ctx context.Context, formID, submittedBy kernel.ID, data map[string]any) (*domain.Entry, error) {
	d, err := s.GetDef(ctx, formID)
	if err != nil {
		return nil, err
	}
	if d.Status != domain.StatusActive {
		return nil, errors.New(errors.KindForbidden, "lcform.form_archived", "表单已归档，不能提交")
	}
	clean, err := validateData(d.Fields, data)
	if err != nil {
		return nil, err
	}
	e := &domain.Entry{
		ID: kernel.NewID(), FormID: formID, Data: clean,
		SubmittedBy: submittedBy, CreatedAt: s.clock.Now(),
	}
	if err := s.repo.CreateEntry(ctx, e); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "lcform.submit_failed", "提交失败", err)
	}
	_ = s.bus.Publish(ctx, "lcform.entry_submitted", map[string]any{"form_id": formID, "entry_id": e.ID, "by": submittedBy})
	return e, nil
}

func (s *Service) ListEntries(ctx context.Context, formID kernel.ID, search string, page kernel.Pagination) (*kernel.Page[*domain.Entry], error) {
	if _, err := s.GetDef(ctx, formID); err != nil {
		return nil, err
	}
	p := page.Normalize()
	rows, total, err := s.repo.ListEntries(ctx, domain.EntryFilter{FormID: formID, Search: strings.TrimSpace(search)}, p.PageSize, p.Offset())
	if err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "lcform.db_error", "操作失败", err)
	}
	return &kernel.Page[*domain.Entry]{Data: rows, Total: total, Page: p.Page, PageSize: p.PageSize}, nil
}

// ExportCSV returns a CSV matrix (header row + one row per entry) for a form. The
// column order follows the form's field schema; submission time is the last column.
func (s *Service) ExportCSV(ctx context.Context, formID kernel.ID) (string, [][]string, error) {
	d, err := s.GetDef(ctx, formID)
	if err != nil {
		return "", nil, err
	}
	entries, err := s.repo.AllEntries(ctx, formID)
	if err != nil {
		return "", nil, errors.Wrap(errors.KindDatabase, "lcform.db_error", "操作失败", err)
	}
	header := make([]string, 0, len(d.Fields)+1)
	for _, f := range d.Fields {
		header = append(header, f.Label)
	}
	header = append(header, "提交时间")
	rows := make([][]string, 0, len(entries)+1)
	rows = append(rows, header)
	for _, e := range entries {
		row := make([]string, 0, len(d.Fields)+1)
		for _, f := range d.Fields {
			row = append(row, cellString(e.Data[f.Key]))
		}
		row = append(row, e.CreatedAt.Format(time.RFC3339))
		rows = append(rows, row)
	}
	filename := fmt.Sprintf("%s_entries.csv", d.Code)
	return filename, rows, nil
}

// ---- helpers ----

func normStatus(s string) string {
	if s == domain.StatusArchived {
		return domain.StatusArchived
	}
	return domain.StatusActive
}

// normFields trims, validates types, and ensures unique non-empty keys.
func normFields(in []domain.Field) ([]domain.Field, error) {
	out := make([]domain.Field, 0, len(in))
	seen := map[string]bool{}
	for i, f := range in {
		key := strings.TrimSpace(f.Key)
		label := strings.TrimSpace(f.Label)
		if key == "" {
			key = fmt.Sprintf("field_%d", i+1)
		}
		if label == "" {
			label = key
		}
		if !domain.ValidFieldType(f.Type) {
			return nil, errors.New(errors.KindParam, "lcform.field_type_invalid", "字段类型无效："+f.Type)
		}
		if seen[key] {
			return nil, errors.New(errors.KindParam, "lcform.field_key_dup", "字段标识重复："+key)
		}
		seen[key] = true
		opts := []string{}
		for _, o := range f.Options {
			if o = strings.TrimSpace(o); o != "" {
				opts = append(opts, o)
			}
		}
		out = append(out, domain.Field{Key: key, Label: label, Type: f.Type, Required: f.Required, Options: opts})
	}
	return out, nil
}

// validateData enforces required fields and keeps only known keys.
func validateData(fields []domain.Field, in map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for _, f := range fields {
		v, present := in[f.Key]
		if f.Required && isEmpty(v) {
			return nil, errors.New(errors.KindParam, "lcform.field_required", "必填项未填写："+f.Label)
		}
		if present {
			out[f.Key] = v
		}
	}
	return out, nil
}

func isEmpty(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(t) == ""
	case []any:
		return len(t) == 0
	case bool:
		return false
	default:
		return false
	}
}

func cellString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "是"
		}
		return "否"
	case float64:
		// trim trailing .0 for whole numbers
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case []any:
		parts := make([]string, 0, len(t))
		for _, x := range t {
			parts = append(parts, cellString(x))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", t)
	}
}

// autoCode derives a slug from the name; falls back to a random-ish prefix.
func autoCode(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteByte('_')
		}
	}
	code := strings.Trim(b.String(), "_")
	if code == "" || !codeRe.MatchString(code) {
		code = "form_" + string(kernel.NewID())[:8]
	}
	if len(code) > 40 {
		code = code[:40]
	}
	return code
}

func notFoundOr(err error, code, msg string) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, pgx.ErrNoRows) {
		return errors.New(errors.KindNotFound, code, msg)
	}
	return errors.Wrap(errors.KindDatabase, "lcform.db_error", "操作失败", err)
}
