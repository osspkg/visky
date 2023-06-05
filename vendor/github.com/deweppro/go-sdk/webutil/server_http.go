package webutil

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	application "github.com/deweppro/go-sdk/app"
	"github.com/deweppro/go-sdk/errors"
	"github.com/deweppro/go-sdk/log"
	"github.com/deweppro/go-sdk/netutil"
)

type (
	ConfigHttp struct {
		Addr            string        `yaml:"addr"`
		Network         string        `yaml:"network,omitempty"`
		ReadTimeout     time.Duration `yaml:"read_timeout,omitempty"`
		WriteTimeout    time.Duration `yaml:"write_timeout,omitempty"`
		IdleTimeout     time.Duration `yaml:"idle_timeout,omitempty"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout,omitempty"`
	}

	ServerHttp struct {
		status  int64
		conf    ConfigHttp
		serv    *http.Server
		handler http.Handler

		log log.Logger
		wg  sync.WaitGroup
	}
)

// NewServerHttp create default http server
func NewServerHttp(conf ConfigHttp, handler http.Handler, l log.Logger) *ServerHttp {
	srv := &ServerHttp{
		conf:    conf,
		handler: handler,
		log:     l,
		status:  statusOff,
	}
	srv.validate()
	return srv
}

func (s *ServerHttp) validate() {
	if s.conf.ReadTimeout == 0 {
		s.conf.ReadTimeout = defaultTimeout
	}
	if s.conf.WriteTimeout == 0 {
		s.conf.WriteTimeout = defaultTimeout
	}
	if s.conf.IdleTimeout == 0 {
		s.conf.IdleTimeout = defaultTimeout
	}
	if s.conf.ShutdownTimeout == 0 {
		s.conf.ShutdownTimeout = defaultShutdownTimeout
	}
	if len(s.conf.Network) == 0 {
		s.conf.Network = defaultNetwork
	}
	if _, ok := networkType[s.conf.Network]; !ok {
		s.conf.Network = defaultNetwork
	}
	s.conf.Addr = netutil.CheckHostPort(s.conf.Addr)
}

// Up start http server
func (s *ServerHttp) Up(ctx application.Context) error {
	if !atomic.CompareAndSwapInt64(&s.status, statusOff, statusOn) {
		return errors.Wrapf(errServAlreadyRunning, "starting server on %s", s.conf.Addr)
	}
	s.serv = &http.Server{
		ReadTimeout:  s.conf.ReadTimeout,
		WriteTimeout: s.conf.WriteTimeout,
		IdleTimeout:  s.conf.IdleTimeout,
		Handler:      s.handler,
	}

	nl, err := net.Listen(s.conf.Network, s.conf.Addr)
	if err != nil {
		return err
	}

	s.log.WithFields(log.Fields{
		"ip": s.conf.Addr,
	}).Infof("HTTP server started")

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err = s.serv.Serve(nl); err != nil && err != http.ErrServerClosed {
			s.log.WithFields(log.Fields{
				"err": err.Error(), "ip": s.conf.Addr,
			}).Errorf("HTTP server stopped")
			ctx.Close()
			return
		}
		s.log.WithFields(log.Fields{
			"ip": s.conf.Addr,
		}).Infof("HTTP server stopped")
	}()
	return nil
}

// Down stop http server
func (s *ServerHttp) Down() error {
	if !atomic.CompareAndSwapInt64(&s.status, statusOn, statusOff) {
		return errors.Wrapf(errServAlreadyStopped, "stopping server on %s", s.conf.Addr)
	}
	ctx, cncl := context.WithTimeout(context.Background(), s.conf.ShutdownTimeout)
	defer cncl()
	err := s.serv.Shutdown(ctx)
	s.wg.Wait()
	return err
}
