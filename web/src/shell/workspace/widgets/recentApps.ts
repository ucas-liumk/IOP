// Track recently-opened apps for the 最近访问 widget.
// Bumped by router.beforeEach when the user navigates into /<module>/*.

import { ref } from "vue";

const KEY = "iop.workspace.recentApps.v1";
const MAX = 6;

export interface RecentApp {
  code: string;
  at: number; // epoch ms
}

function load(): RecentApp[] {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return [];
    return (JSON.parse(raw) as RecentApp[]).slice(0, MAX);
  } catch {
    return [];
  }
}

const recent = ref<RecentApp[]>(load());

export function bumpRecent(code: string) {
  if (!code) return;
  const now = Date.now();
  const without = recent.value.filter((r) => r.code !== code);
  recent.value = [{ code, at: now }, ...without].slice(0, MAX);
  try {
    localStorage.setItem(KEY, JSON.stringify(recent.value));
  } catch {}
}

export function useRecentApps() {
  return { recent };
}
