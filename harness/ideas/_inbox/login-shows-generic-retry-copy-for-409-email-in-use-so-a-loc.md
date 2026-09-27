---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# login shows generic retry copy for 409 email_in_use so a locked-out learner retries forever

## Why
The auth 5xx plan makes `POST /api/v1/auth/google` answer `409 {"error":"email_in_use"}` when the learner's Google email already belongs to a different `google_id`. The PWA's `/login` has a dedicated message only for `google_auth_failed`. Every other code, `email_in_use` included, shows the generic "Không đăng nhập được. Thử lại." ("Couldn't sign in. Try again."). The collision is permanent, so the learner retries forever and never learns why. The plan's own *Notes* named this as a frontend follow-up for the reviewer to file.

## Expected output
- `/login` shows a distinct message that does not invite a retry when `ApiError.code === 'email_in_use'` (e.g. "Email này đã gắn với một tài khoản Google khác."), in the page's existing kit error style.
- A unit or e2e test drives a 409 `email_in_use` reply and asserts the copy.

## Evidence
- Plan: `harness/plans/2026-09-25-auth-reports-postgres-and-redis-failures-as-401-and-logs-not.md`, *Notes and open questions*, "Frontend copy for `409 email_in_use`".
- `frontend/pages/login.vue:43-45` — `e.code === 'google_auth_failed' ? … : 'Không đăng nhập được. Thử lại.'`.
- `backend/internal/auth/handler.go` — `case errors.Is(err, ErrEmailTaken): c.JSON(http.StatusConflict, gin.H{"error": "email_in_use"})`.
