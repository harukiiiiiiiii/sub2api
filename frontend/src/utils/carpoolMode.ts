// Commercial pages unavailable in carpool mode. Keep subscription management.
export function isCarpoolRestrictedPath(path: string): boolean {
  const pathname = path.split(/[?#]/)[0] || '/'
  return [
    '/purchase', '/orders', '/payment', '/redeem', '/affiliate',
    '/admin/orders', '/admin/redeem', '/admin/promo-codes', '/admin/affiliates',
    '/auth/wechat/payment', '/model-plaza', '/batch-image', '/custom', '/register'
  ].some(prefix => pathname === prefix || pathname.startsWith(prefix + '/'))
}
