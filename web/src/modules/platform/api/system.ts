import { client } from "@/api/client";

// Platform-console SYSTEM pages API (P3). All routes are tenant-LESS and gated by
// the global is_platform_admin flag + per-resource PlatformAuthz on the backend.

// ============================== Dict (platform) ==============================
// There is NO platform-level dictionary override API — the only platform-reachable
// dictionary surface is the shared public lookup GET /dict/:typeCode (read-only,
// resolves the in-memory seed + any tenant overrides for the *current* context,
// which for the platform console is just the seed). The set of dict types is not
// exposed by an endpoint, so we keep a small known-type catalog here; selecting a
// type fetches its items via the public lookup. This is intentionally read-only.
export interface DictItem { type_code: string; code: string; name: string; sort_order: number; active: boolean }

// Known platform-visible dict types (mirrors the in-memory seed in app.go). Adding
// a seeded type here surfaces it in the platform dict viewer.
export const PLATFORM_DICT_TYPES: string[] = ["plan_level", "report_type"];

export async function lookupDict(typeCode: string): Promise<DictItem[]> {
  const r = await client.get(`/dict/${typeCode}`);
  return r.data?.data?.items ?? [];
}

// ============================== Params (KV) ==============================
export interface PlatformParam {
  key: string;
  value: any;          // arbitrary JSON
  updated_by?: string;
  updated_at: string;
}
export async function listParams(): Promise<PlatformParam[]> {
  const r = await client.get("/platform/params");
  return r.data?.data?.params ?? [];
}
// value is sent as raw JSON (the backend validates json.Valid). Callers pass the
// already-parsed JS value; we wrap it under { value } and let axios serialize.
export async function upsertParam(key: string, value: any) {
  await client.put(`/platform/params/${encodeURIComponent(key)}`, { value });
}
export async function deleteParam(key: string) {
  await client.delete(`/platform/params/${encodeURIComponent(key)}`);
}

// ============================== Notices ==============================
export interface PlatformNotice {
  id: string;
  title: string;
  content: string;
  type: string;        // "notice" | "announcement"
  status: string;      // "draft" | "published" | "withdrawn"
  created_by?: string;
  created_at: string;
}
export type PlatformNoticeStatus = "draft" | "published" | "withdrawn" | "";
export async function listNotices(status: PlatformNoticeStatus = "", page = 1, pageSize = 50): Promise<PlatformNotice[]> {
  const params: Record<string, string | number> = { page, page_size: pageSize };
  if (status) params.status = status;
  const r = await client.get("/platform/notices", { params });
  return r.data?.data?.notices ?? [];
}
export async function createNotice(payload: { title: string; content?: string; type?: string }): Promise<PlatformNotice> {
  const r = await client.post("/platform/notices", payload);
  return r.data?.data as PlatformNotice;
}
export async function updateNotice(id: string, patch: { title: string; content?: string; type?: string }) {
  await client.patch(`/platform/notices/${id}`, patch);
}
export async function publishNotice(id: string) {
  await client.post(`/platform/notices/${id}/publish`);
}
export async function withdrawNotice(id: string) {
  await client.post(`/platform/notices/${id}/withdraw`);
}
export async function deleteNotice(id: string) {
  await client.delete(`/platform/notices/${id}`);
}

// ============================== Logs (oper / login) ==============================
// Both endpoints return public.platform_audit_log rows (PlatformEntry shape) under
// the `logs` key. Paged (page / page_size), bare array (no total) — the UI infers
// "has next" from a full page. Login logs are matched by action ILIKE '%login%'
// and are typically empty (platform login events are not yet persisted — documented
// backend limitation; the view surfaces a `note` hint).
export interface PlatformLogEntry {
  id: string;
  occurred_at: string;
  actor: string;
  actor_role?: string;
  action: string;
  resource: string;
  resource_id: string;
  reason?: string;
  trace_id: string;
  detail: any;
}
export interface PlatformLogFilter {
  actor?: string;
  action?: string;
  from?: string;   // YYYY-MM-DD
  to?: string;     // YYYY-MM-DD
  page?: number;
  pageSize?: number;
}
function logParams(f: PlatformLogFilter): Record<string, string | number> {
  const p: Record<string, string | number> = { page: f.page ?? 1, page_size: f.pageSize ?? 20 };
  if (f.actor) p.actor = f.actor;
  if (f.action) p.action = f.action;
  if (f.from) p.from = f.from;
  if (f.to) p.to = f.to;
  return p;
}
export async function listOperLogs(f: PlatformLogFilter = {}): Promise<PlatformLogEntry[]> {
  const r = await client.get("/platform/operlogs", { params: logParams(f) });
  return r.data?.data?.logs ?? [];
}
export async function listLoginLogs(f: PlatformLogFilter = {}): Promise<PlatformLogEntry[]> {
  const { action, ...rest } = f;  // login endpoint ignores `action` (pinned to login topics)
  void action;
  const r = await client.get("/platform/loginlogs", { params: logParams(rest) });
  return r.data?.data?.logs ?? [];
}

// ============================== Online users (cross-tenant) ==============================
export interface PlatformOnlineSession {
  session_id: string;
  platform_user_id: string;
  username: string;
  tenant_id?: string;
  tenant_name?: string;
  display_name?: string;
  ip_address?: string;
  issued_at: string;
  expires_at: string;
}
export async function listOnlineSessions(): Promise<PlatformOnlineSession[]> {
  const r = await client.get("/platform/online");
  return r.data?.data?.sessions ?? [];
}
export async function kickSession(sid: string) {
  await client.post(`/platform/online/${sid}/kick`);
}

// ============================== Monitor ==============================
export interface DBPoolStats {
  acquired_conns: number;
  idle_conns: number;
  total_conns: number;
  max_conns: number;
  new_conns_count: number;
  acquire_count: number;
  canceled_acquire_count: number;
  empty_acquire_count: number;
}
export interface HealthCheck { ok?: boolean; status?: string; error?: string; latency_ms?: number; [k: string]: any }
export interface MonitorSnapshot {
  db_pool: DBPoolStats;
  counters: Record<string, number>;
  health?: Record<string, HealthCheck>;
  redis?: Record<string, any>;
}
export async function getMonitor(): Promise<MonitorSnapshot> {
  const r = await client.get("/platform/monitor");
  return r.data?.data ?? { db_pool: {} as DBPoolStats, counters: {} };
}

// ============================== Cron jobs ==============================
export interface PlatformJob {
  id: string;
  name: string;
  cron_expr: string;
  handler: string;
  status: string;        // "enabled" | "disabled"
  last_run_at?: string | null;
  next_run_at?: string | null;
  created_at: string;
}
export interface PlatformJobRun {
  id: string;
  job_id: string;
  started_at: string;
  finished_at?: string | null;
  status: string;        // "running" | "success" | "failed"
  detail: string;
}
export async function listJobs(): Promise<PlatformJob[]> {
  const r = await client.get("/platform/jobs");
  return r.data?.data?.jobs ?? [];
}
export async function createJob(payload: { name: string; cron_expr?: string; handler?: string; status?: string }): Promise<PlatformJob> {
  const r = await client.post("/platform/jobs", payload);
  return r.data?.data as PlatformJob;
}
export async function updateJob(id: string, patch: { name?: string; cron_expr?: string; handler?: string; status?: string }) {
  await client.patch(`/platform/jobs/${id}`, patch);
}
export async function deleteJob(id: string) {
  await client.delete(`/platform/jobs/${id}`);
}
export async function runJobNow(id: string): Promise<PlatformJobRun> {
  const r = await client.post(`/platform/jobs/${id}/run`);
  return r.data?.data?.run as PlatformJobRun;
}
export async function listJobRuns(id: string, page = 1, pageSize = 30): Promise<PlatformJobRun[]> {
  const r = await client.get(`/platform/jobs/${id}/runs`, { params: { page, page_size: pageSize } });
  return r.data?.data?.runs ?? [];
}
