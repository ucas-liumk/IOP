// Widget registry — declarative source of truth for all home widgets.
// Adding a new widget = register here + create <WidgetXxx.vue>.

export type WidgetCode =
  | "agenda"
  | "notifications"
  | "todos"
  | "recent"
  | "announcement";

export interface WidgetDef {
  code: WidgetCode;
  name: string;
  description: string;
  sourceApp: { code: string; name: string; color: string };
  icon: string;
  preview: "list" | "badges" | "grid";
}

export const WIDGETS: WidgetDef[] = [
  {
    code: "notifications",
    name: "新通知",
    description: "未读通知与提醒",
    sourceApp: { code: "iop", name: "平台", color: "var(--info)" },
    icon: "M12 22a2 2 0 0 0 2-2h-4a2 2 0 0 0 2 2zm6-6V11c0-3.07-1.63-5.64-4.5-6.32V4a1.5 1.5 0 1 0-3 0v.68C7.64 5.36 6 7.92 6 11v5l-2 2v1h16v-1l-2-2z",
    preview: "list",
  },
  {
    code: "todos",
    name: "我的待办",
    description: "OKR 中未完成的关键举措",
    sourceApp: { code: "okr", name: "OKR 工作安排", color: "var(--cat-task)" },
    icon: "M9 16.17 4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z",
    preview: "list",
  },
  {
    code: "agenda",
    name: "本周议程",
    description: "本周计划的时间节点与里程碑",
    sourceApp: { code: "okr", name: "OKR 工作安排", color: "var(--cat-goal)" },
    icon: "M19 4h-1V2h-2v2H8V2H6v2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 16H5V10h14v10zm0-12H5V6h14v2z",
    preview: "badges",
  },
  {
    code: "recent",
    name: "最近访问",
    description: "你最近打开过的应用",
    sourceApp: { code: "iop", name: "平台", color: "var(--text-3)" },
    icon: "M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2zm0 18a8 8 0 1 1 8-8 8 8 0 0 1-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z",
    preview: "grid",
  },
  {
    code: "announcement",
    name: "平台公告",
    description: "版本更新、维护公告",
    sourceApp: { code: "iop", name: "平台", color: "var(--warning)" },
    icon: "M18 11v-1l-5-5H7c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h10c1.1 0 2-.9 2-2v-9.18A4 4 0 0 1 18 11zM12 19l-4-4h2.5v-4h3v4H16l-4 4z",
    preview: "list",
  },
];

export function widgetByCode(code: string): WidgetDef | undefined {
  return WIDGETS.find((w) => w.code === code);
}
