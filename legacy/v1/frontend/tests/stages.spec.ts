import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useStagesStore } from '@/stores/stages'
import type { StageDef } from '@/types'

const MOCK_STAGES: StageDef[] = [
  { code: 'submit',    label: '问题提报', branch: null,        seq: 1 },
  { code: 'review',    label: '审核分办', branch: null,        seq: 2 },
  { code: 'propose',   label: '研提举措', branch: null,        seq: 3 },
  { code: 'meeting',   label: '会商研究', branch: 'dispute',   seq: 4 },
  { code: 'arbitrate', label: '争议裁决', branch: 'dispute',   seq: 5 },
  { code: 'consult',   label: '征求意见', branch: 'consensus', seq: 6 },
  { code: 'implement', label: '督导落实', branch: null,        seq: 7 },
  { code: 'evaluate',  label: '评价反馈', branch: null,        seq: 8 },
]

describe('stages store path filtering', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    const store = useStagesStore()
    store.stages = MOCK_STAGES as any
  })

  it('dispute path includes meeting & arbitrate, excludes consult', () => {
    const store = useStagesStore()
    const path = store.pathFor('dispute').map((s) => s.code)
    expect(path).toEqual(['submit', 'review', 'propose', 'meeting', 'arbitrate', 'implement', 'evaluate'])
  })

  it('consensus path includes consult, excludes meeting/arbitrate', () => {
    const store = useStagesStore()
    const path = store.pathFor('consensus').map((s) => s.code)
    expect(path).toEqual(['submit', 'review', 'propose', 'consult', 'implement', 'evaluate'])
  })

  it('null branch path only includes common (treated as consensus)', () => {
    const store = useStagesStore()
    const path = store.pathFor(null).map((s) => s.code)
    // Common-only nodes
    expect(path).toEqual(['submit', 'review', 'propose', 'implement', 'evaluate'])
  })

  it('byCode looks up stage by code', () => {
    const store = useStagesStore()
    expect(store.byCode.meeting.label).toBe('会商研究')
  })
})
