package spotify

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// All users of this client share one gate, including retry cooldowns. The
// channel permits cancellation while queued; the gate protects next as well.
type pacedTransport struct {
	base     http.RoundTripper
	gate     chan struct{}
	interval time.Duration
	next     time.Time
}

func newPacedTransport(base http.RoundTripper, interval time.Duration) *pacedTransport {
	return &pacedTransport{base: base, gate: make(chan struct{}, 1), interval: interval}
}

func waitUntil(ctx context.Context, until time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(time.Until(until))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (t *pacedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		select {
		case <-r.Context().Done():
			return nil, r.Context().Err()
		case t.gate <- struct{}{}:
		}
		if err := waitUntil(r.Context(), t.next); err != nil {
			<-t.gate
			return nil, err
		}
		t.next = time.Now().Add(t.interval)
		resp, err := t.base.RoundTrip(r)
		if err != nil {
			<-t.gate
			return nil, err
		}
		limited := resp.StatusCode == http.StatusTooManyRequests
		if limited {
			until := retryAt(resp.Header.Get("Retry-After"), time.Now())
			if until.After(t.next) {
				t.next = until
			}
		}
		<-t.gate
		if !limited || attempt >= 2 || r.Method != http.MethodGet {
			return resp, nil
		}
		// Do not retain a connection or a large error body while waiting.
		resp.Body.Close()
	}
}

func retryAt(value string, now time.Time) time.Time {
	if seconds, err := strconv.ParseInt(value, 10, 32); err == nil && seconds >= 0 {
		return now.Add(time.Duration(seconds) * time.Second)
	}
	if date, err := http.ParseTime(value); err == nil {
		return date
	}
	return now.Add(5 * time.Second)
}
