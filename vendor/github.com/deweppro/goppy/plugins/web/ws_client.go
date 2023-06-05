package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	context2 "github.com/deweppro/go-sdk/context"
	"github.com/deweppro/go-sdk/errors"
	"github.com/deweppro/go-sdk/log"
	"github.com/deweppro/goppy/plugins"
	"github.com/gorilla/websocket"
)

func WithWebsocketClient() plugins.Plugin {
	return plugins.Plugin{
		Inject: func(l log.Logger) (*wscProvider, WebsocketClient) {
			ctx, cncl := context.WithCancel(context.Background())
			c := &wscProvider{
				connects: make(map[string]WebsocketClientConn),
				log:      l,
				ctx:      ctx,
				cancel:   cncl,
			}
			return c, c
		},
	}
}

type (
	wscProvider struct {
		connects map[string]WebsocketClientConn

		cancel context.CancelFunc
		ctx    context.Context

		mux sync.RWMutex
		wg  sync.WaitGroup

		log log.Logger
	}

	WebsocketClient interface {
		Create(ctx context.Context, url string, opts ...func(WebsocketClientOption)) (WebsocketClientConn, error)
	}
)

func (v *wscProvider) Up() error {
	return nil
}

func (v *wscProvider) Down() error {
	v.cancel()
	v.wg.Wait()
	return nil
}

func (v *wscProvider) addConn(cc WebsocketClientConn) {
	v.wg.Add(1)
	lock(&v.mux, func() {
		v.connects[cc.ConnectID()] = cc
	})
}

func (v *wscProvider) delConn(cid string) {
	lock(&v.mux, func() {
		delete(v.connects, cid)
	})
	v.wg.Done()
}

func (v *wscProvider) errLog(cid string, err error, msg string, args ...interface{}) {
	if err == nil {
		return
	}
	v.log.WithFields(log.Fields{
		"cid": cid,
		"err": err.Error(),
	}).Errorf(msg, args...)
}

func (v *wscProvider) Create(
	ctx context.Context, url string,
	opts ...func(WebsocketClientOption),
) (WebsocketClientConn, error) {
	cc := newWSCConnect(url, v.errLog, ctx, v.ctx, opts)

	cc.OnClose(func(cid string) {
		v.delConn(cid)
	})
	cc.OnOpen(func(cid string) {
		v.addConn(cc)
	})

	return cc, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type (
	wscConn struct {
		status int64
		cid    string

		url     string
		headers http.Header
		conn    *websocket.Conn

		sendC  chan []byte
		events map[uint]WebsocketClientHandler

		ctx    context.Context
		cancel context.CancelFunc

		onOpen, onClose []func(cid string)
		erw             func(cid string, err error, msg string, args ...interface{})

		cm sync.RWMutex
		em sync.RWMutex
	}

	WebsocketClientOption interface {
		Header(key string, value string)
	}

	WebsocketClientHandler func(d WebsocketEventer, c WebsocketClientProcessor) error

	WebsocketClientProcessor interface {
		ConnectID() string
		OnClose(cb func(cid string))
		Encode(eventID uint, in interface{})
		EncodeEvent(event WebsocketEventer, in interface{})
	}

	WebsocketClientConn interface {
		ConnectID() string
		Event(call WebsocketClientHandler, eid ...uint)
		Encode(id uint, in interface{})
		Close()
		Run() error
	}
)

func newWSCConnect(
	url string,
	erw func(cid string, err error, msg string, args ...interface{}),
	ctx1, ctx2 context.Context,
	opts []func(WebsocketClientOption),
) *wscConn {
	ctx, cancel := context2.Combine(ctx1, ctx2)
	cc := &wscConn{
		status:  off,
		url:     url,
		headers: make(http.Header),
		sendC:   make(chan []byte, 128),
		events:  make(map[uint]WebsocketClientHandler, 128),
		ctx:     ctx,
		cancel:  cancel,
		onClose: make([]func(string), 0),
		erw:     erw,
	}

	for _, opt := range opts {
		opt(cc)
	}

	return cc
}

func (v *wscConn) ConnectID() string {
	return v.cid
}

func (v *wscConn) connect() *websocket.Conn {
	return v.conn
}

func (v *wscConn) cancelFunc() context.CancelFunc {
	return v.cancel
}

func (v *wscConn) done() <-chan struct{} {
	return v.ctx.Done()
}

func (v *wscConn) errLog(cid string, err error, msg string, args ...interface{}) {
	v.erw(cid, err, msg, args...)
}

func (v *wscConn) OnClose(cb func(cid string)) {
	lock(&v.cm, func() {
		v.onClose = append(v.onClose, cb)
	})
}

func (v *wscConn) OnOpen(cb func(cid string)) {
	lock(&v.cm, func() {
		v.onOpen = append(v.onOpen, cb)
	})
}

func (v *wscConn) Header(key string, value string) {
	lock(&v.cm, func() {
		v.headers.Set(key, value)
	})
}

func (v *wscConn) Event(call WebsocketClientHandler, eid ...uint) {
	lock(&v.em, func() {
		for _, i := range eid {
			v.events[i] = call
		}
	})
}

func (v *wscConn) getEventHandler(id uint) (h WebsocketClientHandler, ok bool) {
	rwlock(&v.em, func() {
		h, ok = v.events[id]
	})
	return
}

func (v *wscConn) Write(b []byte) {
	if len(b) == 0 {
		return
	}

	select {
	case v.sendC <- b:
	default:
	}
}

func (v *wscConn) dataBus() <-chan []byte {
	return v.sendC
}

func (v *wscConn) Encode(eventID uint, in interface{}) {
	eventModel(func(ev *event) {
		ev.ID = eventID
		ev.Encode(in)
		b, err := json.Marshal(ev)
		if err != nil {
			v.errLog(v.ConnectID(), err, "[ws] encode message: %d", eventID)
			return
		}
		v.Write(b)
	})
}

func (v *wscConn) EncodeEvent(e WebsocketEventer, in interface{}) {
	eventModel(func(ev *event) {
		ev.ID = e.EventID()
		ev.UID = e.UniqueID()
		ev.Encode(in)
		b, err := json.Marshal(ev)
		if err != nil {
			v.errLog(v.ConnectID(), err, "[ws] encode message: %d", e.EventID())
			return
		}
		v.Write(b)
	})
}

func (v *wscConn) dataHandler(b []byte) {
	eventModel(func(ev *event) {
		var (
			err error
			msg string
		)
		defer func() {
			if err != nil {
				v.errLog(v.ConnectID(), err, "[ws] "+msg)
			}
		}()
		if err = json.Unmarshal(b, ev); err != nil {
			msg = "decode message"
			return
		}
		call, ok := v.getEventHandler(ev.EventID())
		if !ok {
			return
		}
		err = call(ev, v)
		if err != nil {
			ev.Error(err)
			bb, er := json.Marshal(ev)
			if er != nil {
				msg = fmt.Sprintf("[ws] call event handler: %d", ev.EventID())
				err = errors.Wrap(err, er)
				return
			}
			err = nil
			v.Write(bb)
			return
		}
	})
}

func (v *wscConn) Close() {
	if !atomic.CompareAndSwapInt64(&v.status, on, down) {
		return
	}
	v.cancel()
}

func (v *wscConn) Run() (err error) {
	if !atomic.CompareAndSwapInt64(&v.status, off, on) {
		return errOneOpenConnect
	}

	var resp *http.Response

	if v.conn, resp, err = websocket.DefaultDialer.Dial(v.url, v.headers); err != nil {
		atomic.CompareAndSwapInt64(&v.status, on, off)
		v.errLog(v.ConnectID(), err, "open connect [%s]", v.url)
		return err
	} else {
		v.cid = resp.Header.Get("Sec-WebSocket-Accept")
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			v.errLog(v.ConnectID(), err, "close body connect [%s]", v.url)
		}
	}()

	rwlock(&v.cm, func() {
		for _, fn := range v.onOpen {
			fn(v.ConnectID())
		}
	})

	setupPingPong(v.connect())
	go pumpWrite(v)
	pumpRead(v)

	rwlock(&v.cm, func() {
		for _, fn := range v.onClose {
			fn(v.ConnectID())
		}
	})

	return nil
}
