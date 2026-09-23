package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// ErrSubscriptionGone means the push service answered 404 or 410: the
// browser unsubscribed or the endpoint expired. The worker deletes the row.
var ErrSubscriptionGone = errors.New("notify: subscription gone")

// Payload is what the PWA's service worker receives (it shows a notification
// and opens URL on click).
type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// DefaultPayload is the daily reminder.
var DefaultPayload = Payload{
	Title: "Time to practice English",
	Body:  "Your 30-minute session is waiting — keep your plant alive.",
	URL:   "/",
}

// PushTTL is how long the push service keeps an undelivered reminder. An
// hour: a reminder about "now" is noise by tomorrow.
const PushTTL = int(time.Hour / time.Second)

// Sender delivers one payload to one subscription.
type Sender interface {
	Send(ctx context.Context, sub Subscription, p Payload) error
}

// WebPushSender is the real Sender (RFC 8291 encryption + RFC 8292 VAPID via
// webpush-go). HTTPClient is an interface so tests can point it anywhere;
// the subscription endpoint is the URL, so httptest needs no base-URL plumbing.
type WebPushSender struct {
	PublicKey  string
	PrivateKey string
	Subscriber string // VAPID `sub`: a mailto: or https: contact
	HTTPClient webpush.HTTPClient
	TTL        int
}

// NewWebPushSender validates the key pair is present (spec §9 VAPID_*).
func NewWebPushSender(publicKey, privateKey, subscriber string) (*WebPushSender, error) {
	if publicKey == "" || privateKey == "" {
		return nil, errors.New("notify: VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY are both required")
	}
	return &WebPushSender{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Subscriber: subscriber,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		TTL:        PushTTL,
	}, nil
}

func (s *WebPushSender) Send(ctx context.Context, sub Subscription, p Payload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("notify: encoding payload: %w", err)
	}
	resp, err := webpush.SendNotificationWithContext(ctx, body,
		&webpush.Subscription{Endpoint: sub.Endpoint, Keys: webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth}},
		&webpush.Options{
			HTTPClient:      s.HTTPClient,
			Subscriber:      s.Subscriber,
			VAPIDPublicKey:  s.PublicKey,
			VAPIDPrivateKey: s.PrivateKey,
			TTL:             s.TTL,
			Urgency:         webpush.UrgencyNormal,
		})
	if err != nil {
		return fmt.Errorf("notify: sending push: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return fmt.Errorf("%w: push service returned %d", ErrSubscriptionGone, resp.StatusCode)
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return fmt.Errorf("notify: push service returned %d", resp.StatusCode)
	}
	return nil
}

var _ Sender = (*WebPushSender)(nil)
