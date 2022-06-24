package main

import (
	"net/http"
	"sync"
	"time"
	"github.com/rs/xid"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter *rate.Limiter
	lastSeen time.Time
}

var visitors = make(map[string]*visitor)
var mu sync.Mutex

func init() {
	go cleanupVisitors()
}

func getVisitor(id string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[id]

	if !exists {
		limiter := rate.NewLimiter(1,3)
		visitors[id] = &visitor{limiter, time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for id, visitor := range visitors {
			if time.Since(visitor.lastSeen) > 3*time.Minute {
				delete(visitors, id)
			}
		}
		mu.Unlock()
	}
}

func limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie("tinywhack-cookie")
		if cookie == nil {
			id := xid.New()
			expiry := time.Now().Add(10000 * time.Minute)
			cookie = &http.Cookie{Name: "tinywhack-cookie", Value: id.String(), Expires: expiry}
			http.SetCookie(w, cookie)
		}

		limiter := getVisitor(cookie.Value)

		if limiter.Allow() == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
