package webutil

import (
	"net/http"

	"github.com/deweppro/go-sdk/log"
)

// RecoveryMiddleware recovery go panic and write to log
func RecoveryMiddleware(l log.Logger) func(
	func(http.ResponseWriter, *http.Request),
) func(http.ResponseWriter, *http.Request) {
	return func(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					if l != nil {
						l.WithFields(log.Fields{"err": err}).Errorf("Recovered")
					}
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			f(w, r)
		}
	}
}
