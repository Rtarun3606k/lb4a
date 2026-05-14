package ratelimmiter

import (
	"lb4a/types"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var visitorLimiters sync.Map

func checkRateLimit(ip string) *rate.Limiter {
	limiter, exisist := visitorLimiters.Load(ip)

	if !exisist {

		rps := types.GetConfig().RateLimit.RequestPerSecond
		burst := types.GetConfig().RateLimit.Burst

		if rps == 0 {
			rps = 10.0 // Default to 10 req/sec
		}
		if burst == 0 {
			burst = 20 // Default to 20 burst
		}
		newLimiter := rate.NewLimiter(rate.Limit(rps), burst)

		visitorLimiters.Store(ip, newLimiter)
		return newLimiter
	}

	return limiter.(*rate.Limiter)

}

func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		ip := strings.Split(r.RemoteAddr, ":")[0]

		limiter := checkRateLimit(ip)

		if !limiter.Allow() {
			duration := time.Since(startTime).String()
			http.Error(w, "429 too many request slow down ", http.StatusTooManyRequests)
			types.Log.Warn("Access Log", "method", r.Method,
				"path", r.URL.Path,
				"status", http.StatusTooManyRequests, // 429
				"duration", duration,
				"client_ip", ip,
			)
			return
		}

		next(w, r)
	}

}
