package domain

import "github.com/leo/iop/server/internal/shared/errors"

var (
	ErrInvalidLevel   = errors.New(errors.KindParam, "okr.plan.invalid_level", "无效的计划级别")
	ErrInvalidPeriod  = errors.New(errors.KindParam, "okr.plan.invalid_period", "时段不合法 (start >= end)")
	ErrEmptyTitle     = errors.New(errors.KindParam, "okr.plan.empty_title", "标题不能为空")
	ErrInvalidStatus  = errors.New(errors.KindBusiness, "okr.plan.invalid_status_transition", "状态转换不合法")
	ErrWeightOverflow = errors.New(errors.KindBusiness, "okr.plan.weight_overflow", "条目权重之和不能超过 100")
	ErrChildOutside   = errors.New(errors.KindBusiness, "okr.plan.child_period_outside_parent", "子计划时段必须在父计划范围内")
	ErrChildLevel     = errors.New(errors.KindBusiness, "okr.plan.invalid_child_level", "子计划级别必须比父计划窄")
	ErrPlanClosed     = errors.New(errors.KindBusiness, "okr.plan.closed", "已关闭的计划不可修改")
	ErrPlanNotFound   = errors.New(errors.KindNotFound, "okr.plan.not_found", "计划不存在")
	ErrItemNotFound   = errors.New(errors.KindNotFound, "okr.plan.item_not_found", "条目不存在")

	ErrInvalidReportType = errors.New(errors.KindParam, "okr.report.invalid_type", "无效的报告类型")
	ErrDailyPeriodWrong  = errors.New(errors.KindParam, "okr.report.daily_period_wrong", "日报时段必须为 1 天")
	ErrWeeklyPeriodWrong = errors.New(errors.KindParam, "okr.report.weekly_period_wrong", "周报时段必须跨 7 天 (周一→周日)")
	ErrReportAlreadySent = errors.New(errors.KindConflict, "okr.report.already_submitted", "该时段报告已提交")
	ErrReportNotFound    = errors.New(errors.KindNotFound, "okr.report.not_found", "报告不存在")
)
