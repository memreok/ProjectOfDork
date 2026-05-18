package handlers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimitBucket struct {
	count      int
	windowEnds time.Time
}

var (
	rateLimitMu      sync.Mutex
	rateLimitBuckets = map[string]rateLimitBucket{}
)

func allowRequest(r *http.Request, scope string, maxRequests int, window time.Duration) bool {
	now := time.Now()
	key := scope + ":" + clientIP(r)

	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	for bucketKey, bucket := range rateLimitBuckets {
		if now.After(bucket.windowEnds.Add(window)) {
			delete(rateLimitBuckets, bucketKey)
		}
	}

	bucket := rateLimitBuckets[key]
	if bucket.windowEnds.IsZero() || now.After(bucket.windowEnds) {
		rateLimitBuckets[key] = rateLimitBucket{
			count:      1,
			windowEnds: now.Add(window),
		}
		return true
	}

	if bucket.count >= maxRequests {
		return false
	}

	bucket.count++
	rateLimitBuckets[key] = bucket
	return true
}

func clientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		return strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
