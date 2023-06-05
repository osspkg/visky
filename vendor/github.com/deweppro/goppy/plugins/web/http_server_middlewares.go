package web

import (
	"net/http"
	"sync/atomic"
)

// Middleware type of middleware
type Middleware func(func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request)

// ThrottlingMiddleware limits active requests
func ThrottlingMiddleware(max int64) Middleware {
	var i int64
	return func(call func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			if atomic.LoadInt64(&i) >= max {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			atomic.AddInt64(&i, 1)
			call(w, r)
			atomic.AddInt64(&i, -1)
		}
	}
}
