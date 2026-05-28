import { client } from "@/api/client";

export interface Plan {
  id: string;
  level: "year" | "half_year" | "month" | "week";
  title: string;
  period: { Start: string; End: string };
  status: "draft" | "active" | "closed";
  items: PlanItem[];
  owner: string;
}

export interface PlanItem {
  ID: string;
  Title: string;
  Weight: number;
  ProgressPct: number;
  ProgressNote: string;
  Status: string;
  SortOrder: number;
}

export interface Report {
  id: string;
  type: "daily" | "weekly";
  owner: string;
  period: { Start: string; End: string };
  summary: string;
  entries: ReportEntry[];
  submitted_at: string;
}

export interface ReportEntry {
  ID: string;
  Title: string;
  Detail: string;
  ProgressNote: string;
  PlanItemID?: string;
  SortOrder: number;
}

export async function listPlans(level?: string): Promise<Plan[]> {
  const res = await client.get("/plans", { params: level ? { level } : {} });
  return res.data.data?.plans ?? [];
}

export async function createPlan(payload: {
  level: string; from: string; to: string; title: string; parent_id?: string;
}): Promise<Plan> {
  const res = await client.post("/plans", payload);
  return res.data.data;
}

export async function getPlan(id: string): Promise<Plan> {
  const res = await client.get(`/plans/${id}`);
  return res.data.data;
}

export async function addItem(planId: string, title: string, weight: number) {
  const res = await client.post(`/plans/${planId}/items`, { title, weight });
  return res.data.data;
}

export async function completeItem(planId: string, itemId: string, note: string) {
  const res = await client.patch(`/plans/${planId}/items/${itemId}/complete`, { note });
  return res.data.data;
}

export async function closePlan(id: string) {
  await client.post(`/plans/${id}/close`);
}

export async function submitDaily(payload: { day: string; summary: string; entries: any[] }) {
  const res = await client.post("/reports/daily", payload);
  return res.data.data as Report;
}

export async function submitWeekly(payload: { week_contains: string; summary: string; entries: any[] }) {
  const res = await client.post("/reports/weekly", payload);
  return res.data.data as Report;
}

export async function listReports(type?: string): Promise<Report[]> {
  const res = await client.get("/reports", { params: type ? { type } : {} });
  return res.data.data?.reports ?? [];
}

export async function getReport(id: string) {
  const res = await client.get(`/reports/${id}`);
  return res.data.data;
}

export async function commentReport(id: string, body: string) {
  await client.post(`/reports/${id}/comments`, { body });
}

export interface RollupRow {
  member_id: string;
  owner_name: string;
  department: string;
  submitted: boolean;
  summary?: string;
}

export async function rollupWeekly(week?: string): Promise<RollupRow[]> {
  const res = await client.get("/rollups/weekly", { params: week ? { week } : {} });
  return res.data.data?.rows ?? [];
}
