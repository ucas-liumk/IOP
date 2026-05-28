import { describe, it, expect } from 'vitest'
import { priorityMeta, statusMeta, colorOf, daysBetween, formatDate } from '@/utils/helpers'

describe('helpers', () => {
  it('priorityMeta returns label & class', () => {
    expect(priorityMeta('critical')).toEqual({ label: '紧急', cls: 'badge-danger' })
    expect(priorityMeta('high')).toEqual({ label: '重要', cls: 'badge-warning' })
    expect(priorityMeta('normal').label).toBe('一般')
  })

  it('statusMeta maps backend codes', () => {
    expect(statusMeta('meeting').label).toBe('会商中')
    expect(statusMeta('done').cls).toBe('badge-success')
    expect(statusMeta('overdue').cls).toBe('badge-danger')
  })

  it('colorOf is deterministic per name', () => {
    expect(colorOf('陈雨晴')).toBe(colorOf('陈雨晴'))
    expect(colorOf('陈')).toBeDefined()
  })

  it('daysBetween handles past/future', () => {
    const past = new Date(Date.now() - 86_400_000 * 3).toISOString().slice(0, 10)
    const fut  = new Date(Date.now() + 86_400_000 * 5).toISOString().slice(0, 10)
    expect(daysBetween(past)).toBeLessThanOrEqual(-2)
    expect(daysBetween(fut)).toBeGreaterThanOrEqual(4)
  })

  it('formatDate trims to length', () => {
    expect(formatDate('2026-05-24 12:00:00')).toBe('2026-05-24')
    expect(formatDate(null)).toBe('—')
    expect(formatDate('2026-05-24 12:00:00', 16)).toBe('2026-05-24 12:00')
  })
})
