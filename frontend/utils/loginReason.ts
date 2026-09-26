/**
 * Maps `/login`'s `?reason=` query param to the sentence shown above the
 * Google button (design `harness/designs/stay-signed-in.md` §7). Anything
 * that is not the one reason the middleware sets — an unknown value, a
 * repeated param (an array), or none at all — renders nothing.
 */
export function loginNotice(reason: unknown): string | null {
  if (reason === 'expired') return 'Phiên đã hết hạn sau 24 giờ không hoạt động. Đăng nhập lại để tiếp tục.'
  return null
}
