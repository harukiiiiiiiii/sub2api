import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import GroupQuotaReset from '../GroupQuotaReset.vue'

const mocks = vi.hoisted(() => ({ list: vi.fn(), reset: vi.fn(), success: vi.fn(), warning: vi.fn() }))
vi.mock('@/api/admin/subscriptions', () => ({ listByGroup: mocks.list, resetGroupQuota: mocks.reset }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: mocks.success, showWarning: mocks.warning }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

function createWrapper() {
  return mount(GroupQuotaReset, {
    props: { group: { id: 7, name: 'DPT' } },
    global: { stubs: { BaseDialog: { props: ['show'], template: '<div v-if="show"><slot/><slot name="footer"/></div>' } } }
  })
}

describe('GroupQuotaReset', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.list.mockResolvedValue({ total: 4, items: [] })
    mocks.reset.mockResolvedValue({ reset_count: 4, cache_warnings: 0 })
  })

  it('requires a window selection and resets the selected group only after confirmation', async () => {
    const wrapper = createWrapper()
    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(mocks.list).toHaveBeenCalledWith(7, 1, 1)
    expect(mocks.reset).not.toHaveBeenCalled()
    const confirm = wrapper.findAll('button').at(-1)!
    expect(confirm.attributes('disabled')).toBeDefined()
    await wrapper.find('input[name="weekly"]').setValue(true)
    await confirm.trigger('click')
    await flushPromises()
    expect(mocks.reset).toHaveBeenCalledWith(7, { daily: false, weekly: true, monthly: false }, expect.any(String))
    expect(wrapper.emitted('reset')).toHaveLength(1)
  })

  it('retains the idempotency key after an ambiguous failure and prevents double submission', async () => {
    const wrapper = createWrapper()
    await wrapper.find('button').trigger('click')
    await flushPromises()
    await wrapper.find('input[name="monthly"]').setValue(true)
    mocks.reset.mockRejectedValueOnce(new Error('Network error'))
    await wrapper.findAll('button').at(-1)!.trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="alert"]').text()).toBe('Network error')
    const key = mocks.reset.mock.calls[0][2]
    let finish!: (result: { reset_count: number; cache_warnings: number }) => void
    mocks.reset.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    const confirm = wrapper.findAll('button').at(-1)!
    await confirm.trigger('click')
    await confirm.trigger('click')
    expect(mocks.reset).toHaveBeenCalledTimes(2)
    expect(mocks.reset.mock.calls[1][2]).toBe(key)
    finish({ reset_count: 4, cache_warnings: 1 })
    await flushPromises()
    expect(mocks.warning).toHaveBeenCalled()
  })

  it('does not reset an empty group or a group whose count failed to load', async () => {
    for (const result of ['empty', 'error']) {
      if (result === 'empty') mocks.list.mockResolvedValueOnce({ total: 0, items: [] })
      else mocks.list.mockRejectedValueOnce(new Error('Load failed'))
      const wrapper = createWrapper()
      await wrapper.find('button').trigger('click')
      await flushPromises()
      await wrapper.find('input[name="daily"]').setValue(true)
      expect(wrapper.findAll('button').at(-1)!.attributes('disabled')).toBeDefined()
      wrapper.unmount()
    }
    expect(mocks.reset).not.toHaveBeenCalled()
  })
})
