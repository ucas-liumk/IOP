import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AvatarBadge from '@/components/AvatarBadge.vue'

describe('AvatarBadge', () => {
  it('renders first char of name', () => {
    const w = mount(AvatarBadge, { props: { name: '陈雨晴' } })
    expect(w.text()).toBe('陈')
  })
  it('honors explicit color', () => {
    const w = mount(AvatarBadge, { props: { name: '李', color: '#ff0000' } })
    const style = (w.attributes('style') || '').toLowerCase()
    expect(style).toMatch(/background:\s*(#ff0000|rgb\(255,\s*0,\s*0\))/)
  })
})
