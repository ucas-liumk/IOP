# 更新日志 / Changelog

本项目所有重要变更都记录在此文件。

All notable changes to this project are documented in this file.

格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- （在此记录尚未发布的新增内容 / Track upcoming additions here.）

### Changed

### Fixed

## [0.1.0] - 2026-05-30

首个公开发布版本：企业多租户 B 端办公平台基座（Go 后端 + Vue 3 前端），「基座 + 可插拔模块」架构。

First public release: a multi-tenant B2B office-platform framework (Go backend + Vue 3 frontend) built around a "core + pluggable modules" architecture.

### Added

- **多租户 schema 隔离 / Multi-tenant schema isolation** — 租户级数据访问经由 `tenantdb.TenantDB.Transaction` 自动 `SET LOCAL search_path`；命门隔离回归测试保障无跨租户数据可达。Tenant-scoped access via auto `SET LOCAL search_path`, with keystone isolation regression tests.
- **JWT + RBAC 鉴权 / Auth** — 基于 JWT 的登录态与基于角色的权限控制；资源×动作权限模型，角色编辑器。JWT-based sessions and role-based access control with a resource × action permission model.
- **注册-审批流程 / Registration-approval flow** — 租户注册申请、平台侧审批落地。Tenant registration applications with platform-side approval.
- **平台 / 租户双角色后台 / Platform & tenant admin console** — 平台管理与租户管理双角色门控的后台管理界面（用户、租户、权限、应用等）。Dual-role gated admin console for platform and tenant administration.
- **应用管理与模块注册表 / App management & module registry** — 统一 `Module` 契约、应用中心（AppCenter）安装/卸载、模块脚手架 `scripts/new-module.sh`，前端路由按 `manifest.ts` 自动发现。Unified `Module` contract, AppCenter install/uninstall, module scaffolding, and auto-discovered frontend routes.
- **OKR 模块 / OKR module** — 完整的 OKR 有界上下文与前端流程，作为可插拔业务模块示例。A complete OKR bounded context with frontend flow, serving as the reference pluggable module.
- **个人中心 / Personal center** — 用户个人信息与账户相关的「我的」中心。Per-user personal center ("me").
- **基础设施 / Infrastructure** — PG / Redis / MinIO 接入、`livez`/`readyz`/`version`/`healthz`/`metrics`、统一错误 envelope、事件总线、i18n 与字典、`trace_id` 贯穿日志。

[Unreleased]: https://github.com/leo/iop/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/leo/iop/releases/tag/v0.1.0
