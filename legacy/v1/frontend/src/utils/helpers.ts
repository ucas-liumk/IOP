import type { PriorityCode, StatusCode } from '@/types'

export const priorityMeta = (p: PriorityCode | string) =>
  ({
    critical: { label: '紧急', cls: 'badge-danger' },
    high:     { label: '重要', cls: 'badge-warning' },
    normal:   { label: '一般', cls: 'badge-info' },
    low:      { label: '低',   cls: 'badge' },
  }[p as PriorityCode] || { label: p as string, cls: 'badge' })

export const statusMeta = (s: StatusCode | string) =>
  ({
    pending:    { label: '待办',     cls: 'badge-warning' },
    processing: { label: '办理中',   cls: 'badge-info' },
    meeting:    { label: '会商中',   cls: 'badge-purple' },
    arbitrate:  { label: '裁决中',   cls: 'badge-danger' },
    consulting: { label: '征求意见中', cls: 'badge-teal' },
    done:       { label: '已办结',   cls: 'badge-success' },
    overdue:    { label: '超期',     cls: 'badge-danger' },
  }[s as StatusCode] || { label: s as string, cls: 'badge' })

const COLORS = ['#1e5fd9','#d63838','#7c4ddb','#0fa8a3','#e8920e','#b14fa0','#2a8856','#1a7fb8']
export function colorOf(name: string): string {
  return COLORS[name.charCodeAt(0) % COLORS.length]
}

export function daysBetween(end: string): number {
  return Math.ceil((new Date(end).getTime() - Date.now()) / 86_400_000)
}

export function formatDate(s?: string | null, len = 10): string {
  if (!s) return '—'
  return s.slice(0, len)
}
