import { describe, expect, it } from 'vitest'
import { loginNotice } from '~/utils/loginReason'

describe('loginNotice', () => {
  it('maps expired to the 24-hour sentence and everything else to null', () => {
    expect(loginNotice('expired')).toBe('Phiên đã hết hạn sau 24 giờ không hoạt động. Đăng nhập lại để tiếp tục.')
    for (const other of ['foo', '', ['expired'], undefined, null, 42]) {
      expect(loginNotice(other)).toBeNull()
    }
  })
})
