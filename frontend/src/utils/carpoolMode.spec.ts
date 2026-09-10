import { describe, expect, it } from 'vitest'
import { isCarpoolRestrictedPath } from './carpoolMode'

describe('carpool commercial routes', () => {
  it.each(['/purchase', '/orders/1', '/payment/result?order=1', '/redeem',
    '/affiliate', '/admin/orders', '/admin/redeem', '/admin/promo-codes',
    '/admin/affiliates/users', '/auth/wechat/payment/callback', '/model-plaza', '/batch-image', '/custom/12', '/register'])('blocks %s', path => {
    expect(isCarpoolRestrictedPath(path)).toBe(true)
  })
  it.each(['/subscriptions', '/admin/subscriptions', '/admin/groups',
    '/admin/users', '/keys', '/usage', '/admin/settings', '/orders-extra'])('preserves %s', path => {
    expect(isCarpoolRestrictedPath(path)).toBe(false)
  })
})
