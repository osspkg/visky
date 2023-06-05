package webutil

import (
	"bytes"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/deweppro/go-sdk/app"
	"github.com/deweppro/go-sdk/errors"
	"github.com/deweppro/go-sdk/log"
	"github.com/deweppro/go-sdk/netutil"
	"golang.org/x/sys/unix"
)

type (
	ConfigEpoll struct {
		Addr            string        `yaml:"addr"`
		Network         string        `yaml:"network,omitempty"`
		ReadTimeout     time.Duration `yaml:"read_timeout,omitempty"`
		WriteTimeout    time.Duration `yaml:"write_timeout,omitempty"`
		IdleTimeout     time.Duration `yaml:"idle_timeout,omitempty"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout,omitempty"`
	}

	ServerEpoll struct {
		status   int64
		close    chan struct{}
		wg       sync.WaitGroup
		handler  EpollHandler
		log      log.Logger
		conf     ConfigEpoll
		eof      []byte
		listener net.Listener
		epoll    *epoll
	}
)

func NewServerEpoll(conf ConfigEpoll, handler EpollHandler, eof []byte, l log.Logger) *ServerEpoll {
	return &ServerEpoll{
		status:  statusOff,
		conf:    conf,
		handler: handler,
		log:     l,
		close:   make(chan struct{}),
		eof:     eof,
	}
}

func (s *ServerEpoll) validate() {
	s.conf.Addr = netutil.CheckHostPort(s.conf.Addr)
	if len(s.eof) == 0 {
		s.eof = defaultEOF
	}
}

// Up ...
func (s *ServerEpoll) Up(ctx app.Context) (err error) {
	if !atomic.CompareAndSwapInt64(&s.status, statusOff, statusOn) {
		return errors.Wrapf(errServAlreadyRunning, "starting server on %s", s.conf.Addr)
	}
	s.validate()
	if s.listener, err = net.Listen("tcp", s.conf.Addr); err != nil {
		return
	}
	if s.epoll, err = newEpoll(s.log); err != nil {
		return
	}
	s.log.WithFields(log.Fields{"ip": s.conf.Addr}).Infof("TCP server started")
	s.wg.Add(2)
	go s.connAccept(ctx)
	go s.epollAccept(ctx)
	return
}

// Down ...
func (s *ServerEpoll) Down() error {
	close(s.close)
	err := errors.Wrap(s.epoll.CloseAll(), s.listener.Close())
	s.wg.Wait()
	s.log.WithFields(log.Fields{"ip": s.conf.Addr}).Infof("TCP server stopped")
	return err
}

func (s *ServerEpoll) connAccept(ctx app.Context) {
	defer func() {
		s.wg.Done()
		ctx.Close()
	}()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.close:
				return
			default:
				s.log.WithFields(log.Fields{"err": err.Error()}).Errorf("Epoll conn accept")
				if err0, ok := err.(net.Error); ok && err0.Temporary() {
					time.Sleep(1 * time.Second)
					continue
				}
				return
			}
		}
		if err = s.epoll.AddOrClose(conn); err != nil {
			s.log.WithFields(log.Fields{
				"err": err.Error(), "ip": conn.RemoteAddr().String(),
			}).Errorf("Epoll add conn")
		}
	}
}

func (s *ServerEpoll) epollAccept(ctx app.Context) {
	defer func() {
		s.wg.Done()
		ctx.Close()
	}()
	for {
		select {
		case <-s.close:
			return
		default:
			list, err := s.epoll.Wait()
			switch err {
			case nil:
			case errEpollEmptyEvents:
				continue
			case unix.EINTR:
				continue
			default:
				s.log.WithFields(log.Fields{"err": err.Error()}).Errorf("Epoll accept conn")
				continue
			}

			for _, c := range list {
				c := c
				go func(conn *epollNetItem) {
					defer conn.Await(false)

					if err1 := newEpollConn(conn.Conn, s.handler, s.eof); err1 != nil {
						if err2 := s.epoll.Close(conn); err2 != nil {
							s.log.WithFields(log.Fields{
								"err": err2.Error(), "ip": conn.Conn.RemoteAddr().String(),
							}).Errorf("Epoll add conn")
						}
						if err1 != io.EOF {
							s.log.WithFields(log.Fields{
								"err": err1.Error(), "ip": conn.Conn.RemoteAddr().String(),
							}).Errorf("Epoll bad conn")
						}
					}
				}(c)
			}
		}
	}
}

/**********************************************************************************************************************/

type epollNetItem struct {
	Conn  net.Conn
	await bool
	Fd    int
	mux   sync.RWMutex
}

func (v *epollNetItem) Await(b bool) {
	v.mux.Lock()
	v.await = b
	v.mux.Unlock()
}

func (v *epollNetItem) IsAwait() bool {
	v.mux.RLock()
	is := v.await
	v.mux.RUnlock()
	return is
}

/**********************************************************************************************************************/

type (
	epollNetMap      map[int]*epollNetItem
	epollNetSlice    []*epollNetItem
	epollEventsSlice []unix.EpollEvent

	//epoll ...
	epoll struct {
		fd     int
		conn   epollNetMap
		events epollEventsSlice
		nets   epollNetSlice
		log    log.Logger
		mux    sync.RWMutex
	}
)

const (
	epollEvents          = unix.POLLIN | unix.POLLRDHUP | unix.POLLERR | unix.POLLHUP | unix.POLLNVAL
	epollEventCount      = 100
	epollEventIntervalMS = 500
)

func newEpoll(l log.Logger) (*epoll, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	return &epoll{
		fd:     fd,
		conn:   make(epollNetMap),
		events: make(epollEventsSlice, epollEventCount),
		nets:   make(epollNetSlice, epollEventCount),
		log:    l,
	}, nil
}

// AddOrClose ...
func (v *epoll) AddOrClose(c net.Conn) error {
	fd := netutil.FileDescriptor(c)
	err := unix.EpollCtl(v.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: epollEvents, Fd: int32(fd)})
	if err != nil {
		return errors.Wrap(err, c.Close())
	}
	v.mux.Lock()
	v.conn[fd] = &epollNetItem{Conn: c, Fd: fd}
	v.mux.Unlock()
	return nil
}

func (v *epoll) removeFD(fd int) error {
	return unix.EpollCtl(v.fd, syscall.EPOLL_CTL_DEL, fd, nil)
}

// Close ...
func (v *epoll) Close(c *epollNetItem) error {
	v.mux.Lock()
	defer v.mux.Unlock()
	return v.closeConn(c)
}

func (v *epoll) closeConn(c *epollNetItem) error {
	if err := v.removeFD(c.Fd); err != nil {
		return err
	}
	delete(v.conn, c.Fd)
	return c.Conn.Close()
}

// CloseAll ...
func (v *epoll) CloseAll() (err error) {
	v.mux.Lock()
	defer v.mux.Unlock()

	for _, conn := range v.conn {
		if err0 := v.closeConn(conn); err0 != nil {
			err = errors.Wrap(err, err0)
		}
	}
	v.conn = make(epollNetMap)
	return
}

func (v *epoll) getConn(fd int) (*epollNetItem, bool) {
	v.mux.RLock()
	conn, ok := v.conn[fd]
	v.mux.RUnlock()
	return conn, ok
}

// Wait ...
func (v *epoll) Wait() (epollNetSlice, error) {
	n, err := unix.EpollWait(v.fd, v.events, epollEventIntervalMS)
	if err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, errEpollEmptyEvents
	}

	v.nets = v.nets[:0]
	for i := 0; i < n; i++ {
		fd := int(v.events[i].Fd)
		conn, ok := v.getConn(fd)
		if !ok {
			if err = v.removeFD(fd); err != nil {
				v.log.WithFields(log.Fields{
					"err": err.Error(), "fd": fd,
				}).Errorf("Close fd")
			}
			continue
		}
		if conn.IsAwait() {
			continue
		}
		conn.Await(true)

		switch v.events[i].Events {
		case unix.POLLIN:
			v.nets = append(v.nets, conn)
		default:
			if err = v.Close(conn); err != nil {
				v.log.WithFields(log.Fields{"err": err.Error()}).Errorf("Epoll close connect")
			}
		}
	}

	return v.nets, nil
}

/**********************************************************************************************************************/

var (
	epollBodyPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 1024)
		},
	}

	errInvalidPoolType = errors.New("invalid data type from pool")
)

type EpollHandler func([]byte, io.Writer) error

func newEpollConn(conn io.ReadWriter, handler EpollHandler, eof []byte) error {
	var (
		n   int
		err error
		l   = len(eof)
	)
	b, ok := epollBodyPool.Get().([]byte)
	if !ok {
		return errInvalidPoolType
	}
	defer epollBodyPool.Put(b[:0]) //nolint:staticcheck

	for {
		if len(b) == cap(b) {
			b = append(b, 0)[:len(b)]
		}
		n, err = conn.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if len(b) < l {
			return io.EOF
		}
		if bytes.Equal(eof, b[len(b)-l:]) {
			b = b[:len(b)-l]
			break
		}
	}
	return handler(b, conn)
}
