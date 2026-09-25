package auth

import "errors"

// ErrGoogleRejected means Google refused the code or the access token (any
// 4xx from the token or userinfo endpoint, or a 2xx without the fields sign-in
// needs). The only sensible client response is to restart the consent flow,
// so Handler answers 401. Every other failure is ours and answers 5xx.
var ErrGoogleRejected = errors.New("auth: google rejected the sign-in")

// ErrEmailTaken means users.email already belongs to a different google_id
// (§3.2: email is UNIQUE; the upsert conflicts on google_id only). Nothing
// merges accounts, so Handler answers 409 and logs it.
var ErrEmailTaken = errors.New("auth: email already belongs to another google account")

// ErrSessionStoreUnavailable wraps a Redis failure that is not "key absent".
// Require answers 503 for it and leaves the session alone: a Redis blip must
// not sign anyone out.
var ErrSessionStoreUnavailable = errors.New("auth: session store unavailable")
