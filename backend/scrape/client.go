package scrape

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	BaseURL   = "https://www.robotcombatevents.com"
	UserAgent = "RobotRegistry/1.0 (+on-demand refresh)"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

const (
	defaultMinDelay = 900 * time.Millisecond
	maxRetries      = 6
)

var (
	rngMu sync.Mutex
	rng   = rand.New(rand.NewSource(time.Now().UnixNano()))

	rateMu      sync.Mutex
	nextAllowed time.Time
)

func absoluteURL(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return BaseURL + href
	}
	return BaseURL + "/" + href
}

func fetchDocument(url string) (*goquery.Document, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("empty url")
	}

	// Global throttling: keep a minimum delay between outbound requests across the entire process.
	throttle(defaultMinDelay)

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", UserAgent)

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			// Network-ish failures: backoff and retry.
			sleepBackoff(attempt, 800*time.Millisecond, 10*time.Second)
			continue
		}
		body := resp.Body

		if resp.StatusCode == http.StatusOK {
			doc, err := goquery.NewDocumentFromReader(body)
			_ = body.Close()
			if err != nil {
				lastErr = err
				sleepBackoff(attempt, 800*time.Millisecond, 10*time.Second)
				continue
			}
			return doc, nil
		}

		// Always drain a small amount so we can reuse connections.
		_, _ = io.CopyN(io.Discard, body, 2048)
		_ = body.Close()

		// Respect rate limiting.
		if resp.StatusCode == http.StatusTooManyRequests {
			wait := retryAfter(resp)
			if wait <= 0 {
				wait = 15 * time.Second
			}
			// Add a bit of jitter so we don't hammer in lockstep.
			wait += jitter(250 * time.Millisecond)
			lastErr = fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status)
			time.Sleep(wait)
			// After sleeping, also push out the next globally-allowed request.
			throttle(wait)
			continue
		}

		// Retry on transient server errors.
		if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			lastErr = fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status)
			sleepBackoff(attempt, 800*time.Millisecond, 15*time.Second)
			continue
		}

		// Non-retryable (e.g. 404/403).
		return nil, fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status)
	}

	if lastErr == nil {
		lastErr = errors.New("failed fetching document")
	}
	return nil, lastErr
}

func throttle(minDelay time.Duration) {
	rateMu.Lock()
	now := time.Now()
	if now.Before(nextAllowed) {
		sleep := time.Until(nextAllowed)
		rateMu.Unlock()
		time.Sleep(sleep)
		rateMu.Lock()
		now = time.Now()
	}
	nextAllowed = now.Add(minDelay)
	rateMu.Unlock()
}

func sleepBackoff(attempt int, base time.Duration, max time.Duration) {
	// exponential backoff: base * 2^attempt, clamped
	d := base << attempt
	if d > max {
		d = max
	}
	d += jitter(250 * time.Millisecond)
	time.Sleep(d)
}

func jitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	rngMu.Lock()
	n := rng.Int63n(int64(max) + 1)
	rngMu.Unlock()
	return time.Duration(n)
}

func retryAfter(resp *http.Response) time.Duration {
	// Retry-After can be seconds or an HTTP date.
	h := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if h == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(h); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}
