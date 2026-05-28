import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StageChip from '@/components/StageChip.vue'

describe('StageChip', () => {
  it('renders the stage label', () => {
    const wrapper = mount(StageChip, { props: { stage: 'meeting' } })
    expect(wrapper.text()).toContain('会商研究')
    expect(wrapper.classes()).toContain('stage-bg-meeting')
  })

  it('applies lg size modifier', () => {
    const wrapper = mount(StageChip, { props: { stage: 'arbitrate', size: 'lg' } })
    expect(wrapper.classes()).toContain('lg')
  })
})
