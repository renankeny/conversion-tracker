package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/httprate"
	"github.com/redis/go-redis/v9"
)

var rateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
local windowMs = tonumber(ARGV[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], windowMs)
elseif redis.call("PTTL", KEYS[1]) < 0 then
  redis.call("PEXPIRE", KEYS[1], windowMs)
end
local pttl = redis.call("PTTL", KEYS[1])
return {current, pttl}
`)

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Duration, bool) {
	if rl == nil || rl.rdb == nil || limit <= 0 {
		return true, limit, 0, false
	}
	res, err := rateLimitScript.Run(ctx, rl.rdb, []string{"rl:" + key}, window.Milliseconds()).Slice()
	if err != nil || len(res) < 2 {
		return true, limit, 0, false
	}
	cur, _ := res[0].(int64)
	pttl, _ := res[1].(int64)
	rem := limit - int(cur)
	if rem < 0 {
		rem = 0
	}
	reset := time.Duration(pttl) * time.Millisecond
	if reset < 0 {
		reset = window
	}
	return int(cur) <= limit, rem, reset, true
}

// LimitByKey aplica rate limiting distribuído no Redis com fallback para httprate local se Redis estiver indisponível.
func (rl *RateLimiter) LimitByKey(scope string, limit int, window time.Duration, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	fallback := httprate.Limit(limit, window, httprate.WithKeyFuncs(func(r *http.Request) (string, error) {
		return scope + ":" + keyFn(r), nil
	}))
	return func(next http.Handler) http.Handler {
		fallbackHandler := fallback(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			k := strings.TrimSpace(keyFn(r))
			if k == "" {
				k = requestHostIP(r)
			}
			allowed, rem, reset, redisOK := rl.Allow(r.Context(), fmt.Sprintf("%s:%s", scope, k), limit, window)
			if !redisOK {
				fallbackHandler.ServeHTTP(w, r)
				return
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rem))
			if !allowed {
				retrySec := int(reset.Seconds())
				if retrySec < 1 {
					retrySec = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retrySec))
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// LimitByIP limita requisições por endereço IP real.
func (rl *RateLimiter) LimitByIP(scope string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return rl.LimitByKey(scope, limit, window, requestHostIP)
}

// LimitWebhooks limita requisições de webhooks por tenant/pixel + gateway (nunca bloqueando IPs compartilhados de Hotmart/Kiwify/Cakto).
func (rl *RateLimiter) LimitWebhooks(limit int, window time.Duration) func(http.Handler) http.Handler {
	return rl.LimitByKey("webhooks", limit, window, func(r *http.Request) string {
		q := r.URL.Query()
		tenantKey := strings.TrimSpace(q.Get("acc"))
		if tenantKey == "" {
			tenantKey = strings.TrimSpace(q.Get("pixel"))
		}
		if tenantKey == "" {
			tenantKey = strings.TrimSpace(q.Get("token"))
		}
		if tenantKey == "" {
			tenantKey = "global"
		}
		return r.URL.Path + ":" + tenantKey
	})
}

func requestHostIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

