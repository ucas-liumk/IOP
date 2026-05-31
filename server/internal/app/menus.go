package app

import "github.com/leo/iop/server/internal/shared/module"

// builtinMenus returns the static console navigation for the platform's own
// (non-module) surfaces: the tenant console (/admin/*) and the platform console
// (/platform/*). Module-contributed menus are merged on top via Registry.AllMenus().
//
// These nodes have App="" (built-in, not gated by AppStore enablement). Their Perm
// strings are the visibility gate: a tenant_admin / super_admin (wildcard '*'/'*'
// policy) trivially satisfies all of them, while a scoped role only sees the nodes
// whose resource:action its policies permit.
//
// Icons are single SVG path-data strings (the same convention as Manifest.Icon).
func builtinMenus() []module.MenuNode {
	return []module.MenuNode{
		// ===== Tenant console (/admin/*) =====
		{
			Key: "admin.home", Title: "仪表盘", Path: "/admin", Type: "menu",
			Console: "tenant", Order: 1,
			Icon: "M3 3h7v9H3zM14 3h7v5h-7zM14 12h7v9h-7zM3 16h7v5H3z",
		},
		// 组织管理
		{
			Key: "admin.org", Title: "组织管理", Type: "dir",
			Console: "tenant", Order: 10,
			Icon: "M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2",
		},
		{
			Key: "admin.members", Title: "成员", Path: "/admin/members", Parent: "admin.org",
			Type: "menu", Console: "tenant", Perm: "member:read", Order: 11,
			Icon: "M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2 M9 7a4 4 0 1 0 0 0",
		},
		{
			Key: "admin.departments", Title: "部门", Path: "/admin/departments", Parent: "admin.org",
			Type: "menu", Console: "tenant", Perm: "dept:read", Order: 12,
			Icon: "M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3z",
		},
		{
			Key: "admin.posts", Title: "岗位", Path: "/admin/posts", Parent: "admin.org",
			Type: "menu", Console: "tenant", Perm: "post:read", Order: 13,
			Icon: "M20 7h-3V5a2 2 0 0 0-2-2H9a2 2 0 0 0-2 2v2H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2z",
		},
		{
			Key: "admin.roles", Title: "角色权限", Path: "/admin/roles", Parent: "admin.org",
			Type: "menu", Console: "tenant", Perm: "role:read", Order: 14,
			Icon: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z",
		},
		{
			Key: "admin.registrations", Title: "注册申请", Path: "/admin/registrations", Parent: "admin.org",
			Type: "menu", Console: "tenant", Perm: "registration:read", Order: 15,
			Icon: "M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M20 8v6M23 11h-6",
		},
		// 内容与日志
		{
			Key: "admin.content", Title: "内容与日志", Type: "dir",
			Console: "tenant", Order: 20,
			Icon: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8zM14 2v6h6",
		},
		{
			Key: "admin.notices", Title: "通知公告", Path: "/admin/notices", Parent: "admin.content",
			Type: "menu", Console: "tenant", Perm: "notice:read", Order: 21,
			Icon: "M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 0 1-3.46 0",
		},
		{
			Key: "admin.logs", Title: "操作/登录日志", Path: "/admin/logs", Parent: "admin.content",
			Type: "menu", Console: "tenant", Perm: "log:read", Order: 22,
			Icon: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8zM14 2v6h6M9 13h6M9 17h6",
		},
		{
			Key: "admin.online", Title: "在线用户", Path: "/admin/online", Parent: "admin.content",
			Type: "menu", Console: "tenant", Perm: "online:read", Order: 23,
			Icon: "M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M9 7a4 4 0 1 0 0 0M22 12l-3 3-1.5-1.5",
		},
		// 系统
		{
			Key: "admin.system", Title: "系统", Type: "dir",
			Console: "tenant", Order: 30,
			Icon: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z",
		},
		{
			Key: "admin.dict", Title: "字典管理", Path: "/admin/dict", Parent: "admin.system",
			Type: "menu", Console: "tenant", Perm: "dict:read", Order: 31,
			Icon: "M4 19.5A2.5 2.5 0 0 1 6.5 17H20M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z",
		},
		{
			Key: "admin.settings", Title: "租户设置", Path: "/admin/settings", Parent: "admin.system",
			Type: "menu", Console: "tenant", Perm: "settings:read", Order: 32,
			Icon: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z",
		},
		{
			Key: "admin.apps", Title: "应用管理", Path: "/admin/apps", Parent: "admin.system",
			Type: "menu", Console: "tenant", Perm: "app:read", Order: 33,
			Icon: "M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z",
		},
		{
			Key: "admin.menus", Title: "菜单管理", Path: "/admin/menus", Parent: "admin.system",
			Type: "menu", Console: "tenant", Perm: "menu:read", Order: 34,
			Icon: "M3 12h18M3 6h18M3 18h18",
		},

		// ===== Platform console (/platform/*) =====
		{
			Key: "platform.home", Title: "概览", Path: "/platform", Type: "menu",
			Console: "platform", Order: 1,
			Icon: "M3 3h7v9H3zM14 3h7v5h-7zM14 12h7v9h-7zM3 16h7v5H3z",
		},
		// 治理 — cross-tenant governance.
		{
			Key: "platform.gov", Title: "治理", Type: "dir",
			Console: "platform", Order: 10,
			Icon: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z",
		},
		{
			Key: "platform.organizations", Title: "组织管理", Path: "/platform/organizations", Parent: "platform.gov",
			Type: "menu", Console: "platform", Perm: "org:read", Order: 11,
			Icon: "M3 6h18v15H3zM3 10h18M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2",
		},
		{
			Key: "platform.users", Title: "全局用户", Path: "/platform/users", Parent: "platform.gov",
			Type: "menu", Console: "platform", Perm: "user:read", Order: 12,
			Icon: "M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M9 7a4 4 0 1 0 0 0M23 21v-2a4 4 0 0 0-3-3.87",
		},
		{
			Key: "platform.rbac", Title: "平台角色", Path: "/platform/rbac", Parent: "platform.gov",
			Type: "menu", Console: "platform", Perm: "role:manage", Order: 13,
			Icon: "M3 11h18v11H3zM7 11V7a5 5 0 0 1 10 0v4",
		},
		{
			Key: "platform.registrations", Title: "注册申请", Path: "/platform/registrations", Parent: "platform.gov",
			Type: "menu", Console: "platform", Perm: "org:read", Order: 14,
			Icon: "M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M20 8v6M23 11h-6",
		},
		// 系统 — platform-wide system config (menus / dict / params / notices).
		{
			Key: "platform.system", Title: "系统", Type: "dir",
			Console: "platform", Order: 20,
			Icon: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z",
		},
		{
			Key: "platform.menus", Title: "菜单管理", Path: "/platform/menus", Parent: "platform.system",
			Type: "menu", Console: "platform", Perm: "menu:read", Order: 21,
			Icon: "M3 12h18M3 6h18M3 18h18",
		},
		{
			Key: "platform.dict", Title: "字典管理", Path: "/platform/dict", Parent: "platform.system",
			Type: "menu", Console: "platform", Perm: "dict:read", Order: 22,
			Icon: "M4 19.5A2.5 2.5 0 0 1 6.5 17H20M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z",
		},
		{
			Key: "platform.params", Title: "参数配置", Path: "/platform/params", Parent: "platform.system",
			Type: "menu", Console: "platform", Perm: "param:read", Order: 23,
			Icon: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33",
		},
		{
			Key: "platform.notices", Title: "通知公告", Path: "/platform/notices", Parent: "platform.system",
			Type: "menu", Console: "platform", Perm: "notice:read", Order: 24,
			Icon: "M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 0 1-3.46 0",
		},
		// 监控 — logs / online / health / cron.
		{
			Key: "platform.monitorgrp", Title: "监控", Type: "dir",
			Console: "platform", Order: 30,
			Icon: "M22 12h-4l-3 9L9 3l-3 9H2",
		},
		{
			Key: "platform.logs", Title: "操作/登录日志", Path: "/platform/logs", Parent: "platform.monitorgrp",
			Type: "menu", Console: "platform", Perm: "audit:read", Order: 31,
			Icon: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8zM14 2v6h6M9 13h6M9 17h6",
		},
		{
			Key: "platform.online", Title: "在线用户", Path: "/platform/online", Parent: "platform.monitorgrp",
			Type: "menu", Console: "platform", Perm: "session:read", Order: 32,
			Icon: "M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M9 7a4 4 0 1 0 0 0M22 12l-3 3-1.5-1.5",
		},
		{
			Key: "platform.monitor", Title: "系统监控", Path: "/platform/monitor", Parent: "platform.monitorgrp",
			Type: "menu", Console: "platform", Perm: "monitor:read", Order: 33,
			Icon: "M3 3v18h18M7 14l4-4 3 3 5-5",
		},
		{
			Key: "platform.cron", Title: "定时任务", Path: "/platform/cron", Parent: "platform.monitorgrp",
			Type: "menu", Console: "platform", Perm: "job:read", Order: 34,
			Icon: "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20zM12 6v6l4 2",
		},
	}
}
