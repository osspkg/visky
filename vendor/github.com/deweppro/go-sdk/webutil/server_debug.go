package webutil

import (
	"net/http"
	"net/http/pprof"

	"github.com/deweppro/go-sdk/app"
	"github.com/deweppro/go-sdk/log"
)

// ServerDebug service model
type ServerDebug struct {
	server *ServerHttp
	route  *Router
}

// NewServerDebug init debug service
func NewServerDebug(c ConfigHttp, l log.Logger) *ServerDebug {
	route := NewRouter()
	return &ServerDebug{
		server: NewServerHttp(c, route, l),
		route:  route,
	}
}

// Up start service
func (o *ServerDebug) Up(ctx app.Context) error {
	o.route.Route("/debug/pprof", pprof.Index, http.MethodGet)
	o.route.Route("/debug/pprof/goroutine", pprof.Index, http.MethodGet)
	o.route.Route("/debug/pprof/allocs", pprof.Index, http.MethodGet)
	o.route.Route("/debug/pprof/block", pprof.Index, http.MethodGet)
	o.route.Route("/debug/pprof/heap", pprof.Index, http.MethodGet)
	o.route.Route("/debug/pprof/mutex", pprof.Index, http.MethodGet)
	o.route.Route("/debug/pprof/threadcreate", pprof.Index, http.MethodGet)
	o.route.Route("/debug/pprof/cmdline", pprof.Cmdline, http.MethodGet)
	o.route.Route("/debug/pprof/profile", pprof.Profile, http.MethodGet)
	o.route.Route("/debug/pprof/symbol", pprof.Symbol, http.MethodGet)
	o.route.Route("/debug/pprof/trace", pprof.Trace, http.MethodGet)
	return o.server.Up(ctx)
}

// Down stop service
func (o *ServerDebug) Down() error {
	return o.server.Down()
}
