package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/nofendian17/sbterm/libs/pkg/response"
)

type Middleware func(http.Handler) http.Handler

type Option func(*options)

type options struct {
	ratePerSecond   int
	burst           int
	keyFn           func(*http.Request) string
	cleanupInterval time.Duration
	clientTTL       time.Duration
}

func WithRatePerSecond(r int) Option {
	return func(o *options) { o.ratePerSecond = r }
}

func WithBurst(b int) Option {
	return func(o *options) { o.burst = b }
}

func WithKeyExtractor(fn func(*http.Request) string) Option {
	return func(o *options) { o.keyFn = fn }
}

func WithCleanupInterval(d time.Duration) Option {
	return func(o *options) { o.cleanupInterval = d }
}

func WithClientTTL(d time.Duration) Option {
	return func(o *options) { o.clientTTL = d }
}

type rateLimitClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimit struct {
	limit           rate.Limit
	burst           int
	keyFn           func(*http.Request) string
	cleanupInterval time.Duration
	clientTTL       time.Duration

	mu          sync.Mutex
	clients     map[string]*rateLimitClient
	lastCleanup time.Time
}

func NewRateLimit(opts ...Option) Middleware {
	o := &options{
		ratePerSecond: 10,
		burst:         20,
		keyFn:         clientIP,
		clientTTL:     10 * time.Minute,
	}
	for _, opt := range opts {
		opt(o)
	}
	if o.ratePerSecond < 1 {
		o.ratePerSecond = 10
	}
	if o.burst < 1 {
		o.burst = 20
	}

	rl := &RateLimit{
		limit:           rate.Limit(o.ratePerSecond),
		burst:           o.burst,
		keyFn:           o.keyFn,
		cleanupInterval: o.cleanupInterval,
		clientTTL:       o.clientTTL,
		clients:         make(map[string]*rateLimitClient),
	}

	return rl.Handler
}

func (rl *RateLimit) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := rl.getLimiter(rl.keyFn(r))

		res := limiter.Reserve()
		if !res.OK() {
			response.Error(w, http.StatusTooManyRequests, response.CodeTooManyRequests, "rate limit exceeded")
			return
		}
		if d := res.Delay(); d > 0 {
			res.Cancel()
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(d.Seconds()))))
			response.Error(w, http.StatusTooManyRequests, response.CodeTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimit) getLimiter(key string) *rate.Limiter {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.cleanupExpired(now)

	c, ok := rl.clients[key]
	if !ok {
		c = &rateLimitClient{
			limiter:  rate.NewLimiter(rl.limit, rl.burst),
			lastSeen: now,
		}
		rl.clients[key] = c
		return c.limiter
	}
	c.lastSeen = now
	return c.limiter
}

func (rl *RateLimit) cleanupExpired(now time.Time) {
	if rl.cleanupInterval <= 0 || rl.clientTTL <= 0 {
		return
	}
	if !rl.lastCleanup.IsZero() && now.Sub(rl.lastCleanup) < rl.cleanupInterval {
		return
	}
	rl.lastCleanup = now

	for k, c := range rl.clients {
		if now.Sub(c.lastSeen) > rl.clientTTL {
			delete(rl.clients, k)
		}
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
