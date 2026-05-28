# 事件总线 Topic 注册表

所有发布到 `shared/eventbus` 的事件 topic 都登记在此文件. 新增 topic 必须更新.

## 命名约定

- `<bounded-context>.<event_name>` (snake_case)
- 事件名为 **过去时** (有意义的、已发生的事): `created`, `joined`, `submitted`, …
- payload 字段 snake_case

## 已注册 Topic

| Topic | 发布者 | 订阅者 | Payload (核心字段) |
|---|---|---|---|
| `tenancy.tenant_created` | tenancy.CreateTenant | audit | tenant_id, slug, name |
| `tenancy.tenant_suspended` | tenancy.SuspendTenant | audit | tenant_id, reason |
| `tenancy.tenant_resumed` | tenancy.ResumeTenant | audit | tenant_id |
| `tenancy.tenant_closed` | tenancy.CloseTenant | audit | tenant_id |
| `tenancy.member_joined` | tenancy.JoinMember | audit, notification | tenant_id, member_id, platform_user_id |
| `iam.user_logged_in` | iam.Login (成功) | audit | platform_user_id, session_id, ip |
| `iam.user_logged_out` | iam.Logout | audit | session_id |
| `iam.login_failed` | iam.Login (失败) | audit | email |
| `okr.plan_created` | okr.CreatePlan | audit | plan_id, level, owner, period_start/end, title |
| `okr.plan_item_added` | okr.AddPlanItem | audit | plan_id, item_id, title |
| `okr.plan_item_completed` | okr.CompleteItem | audit | plan_id, item_id, progress |
| `okr.plan_closed` | okr.ClosePlan | audit | plan_id, closed_at |
| `okr.daily_submitted` | okr.SubmitDaily | audit, notification | report_id, type=daily, owner, period_end |
| `okr.weekly_submitted` | okr.SubmitWeekly | audit, notification | report_id, type=weekly, owner, period_end |
| `okr.weekly_overdue` | okr.RemindOverdue (cron) | notification | owner, period_end |

## 版本演进

事件 payload **新增字段无需版本号** (向后兼容).
**改字段类型 / 删字段 = breaking change**, 必须:
1. 新增新 topic (例: `okr.weekly_submitted_v2`)
2. 双发 4 周
3. 通知订阅者迁移
4. 弃用旧 topic

## 调试

某事件被谁订阅了?

```bash
grep -rn 'bus.Subscribe("topic.name"' server/internal/
```

某事件没被处理?

1. 启用 `IOP_LOGGER_LEVEL=debug`
2. 看日志 `"audit write failed"` 或 `"eventbus handler error"`
3. 确认 `audit.Subscribe` 的 topic 列表包含该 topic
