package web

//go:generate easyjson

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

type WebsocketServerOption func(upg websocket.Upgrader)

func WebsocketServerOptionCompression(enable bool) WebsocketServerOption {
	return func(upg websocket.Upgrader) {
		upg.EnableCompression = enable
	}
}

func WebsocketServerOptionBuffer(read, write int) WebsocketServerOption {
	return func(upg websocket.Upgrader) {
		upg.ReadBufferSize, upg.WriteBufferSize = read, write
	}
}

func WithWebsocketServer(options ...WebsocketServerOption) plugins.Plugin {
	return plugins.Plugin{
		Inject: func(l log.Logger) (*wssProvider, WebsocketServer) {
			for _, option := range options {
				option(wsu)
			}
			wsp := newWsServerProvider(l)
			return wsp, wsp
		},
	}
}

type (
	wssProvider struct {
		status  int64
		clients map[string]*wssConn
		events  map[uint]WebsocketServerHandler

		ctx    context.Context
		cancel context.CancelFunc

		cm sync.RWMutex
		em sync.RWMutex

		log log.Logger
	}

	WebsocketServerHandler func(d WebsocketEventer, c WebsocketServerProcessor) error

	WebsocketServer interface {
		Handling(ctx Context)
		Event(call WebsocketServerHandler, eid ...uint)
		Broadcast(t uint, m json.Marshaler)
		CloseAll()
		CountConn() int
	}
)

func newWsServerProvider(l log.Logger) *wssProvider {
	c, cancel := context.WithCancel(context.TODO())

	return &wssProvider{
		status:  off,
		clients: make(map[string]*wssConn),
		events:  make(map[uint]WebsocketServerHandler),
		ctx:     c,
		cancel:  cancel,
		log:     l,
	}
}

func (v *wssProvider) Up() error {
	if !atomic.CompareAndSwapInt64(&v.status, off, on) {
		return errServAlreadyRunning
	}
	return nil
}

// Down hub
func (v *wssProvider) Down() error {
	if !atomic.CompareAndSwapInt64(&v.status, on, off) {
		return errServAlreadyStopped
	}
	v.CloseAll()
	return nil
}

func (v *wssProvider) Broadcast(t uint, m json.Marshaler) {
	eventModel(func(ev *event) {
		ev.ID = t

		b, err := m.MarshalJSON()
		if err != nil {
			v.errLog("*", err, "[ws] Broadcast error")
			return
		}
		ev.Body(b)

		b, err = json.Marshal(ev)
		if err != nil {
			v.errLog("*", err, "[ws] Broadcast error")
			return
		}

		v.cm.RLock()
		for _, c := range v.clients {
			c.Write(b)
		}
		v.cm.RUnlock()
	})
}

func (v *wssProvider) CloseAll() {
	v.cancel()
}

func (v *wssProvider) Event(call WebsocketServerHandler, eid ...uint) {
	lock(&v.em, func() {
		for _, i := range eid {
			v.events[i] = call
		}
	})
}

func (v *wssProvider) addConn(c *wssConn) {
	lock(&v.cm, func() {
		v.clients[c.ConnectID()] = c
	})
}

func (v *wssProvider) delConn(id string) {
	lock(&v.cm, func() {
		delete(v.clients, id)
	})
}

func (v *wssProvider) CountConn() (cc int) {
	rwlock(&v.cm, func() {
		cc = len(v.clients)
	})
	return
}

func (v *wssProvider) getEventHandler(id uint) (h WebsocketServerHandler, ok bool) {
	rwlock(&v.em, func() {
		h, ok = v.events[id]
	})
	return
}

func (v *wssProvider) errLog(cid string, err error, msg string, args ...interface{}) {
	if err == nil {
		return
	}
	v.log.WithFields(log.Fields{
		"cid": cid,
		"err": err.Error(),
	}).Errorf(msg, args...)
}

func (v *wssProvider) Handling(ctx Context) {
	cid := ctx.Header().Get("Sec-Websocket-Key")

	wsc, err := wsu.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		v.errLog(cid, err, "[ws] upgrade")
		ctx.Error(http.StatusBadRequest, err)
		return
	}

	c := newWSSConnect(cid, v.getEventHandler, v.errLog, wsc, ctx.Context(), v.ctx)

	c.OnClose(func(cid string) {
		v.delConn(cid)
	})
	c.OnOpen(func(string) {
		v.addConn(c)
	})

	c.Run()
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type (
	wssConn struct {
		status int64
		cid    string

		conn  *websocket.Conn
		sendC chan []byte

		ctx    context.Context
		cancel context.CancelFunc

		onClose, onOpen []func(cid string)
		erw             func(cid string, err error, msg string, args ...interface{})
		event           func(id uint) (WebsocketServerHandler, bool)

		mux sync.RWMutex
	}

	WebsocketServerProcessor interface {
		ConnectID() string
		OnClose(cb func(cid string))
		OnOpen(cb func(cid string))
		Encode(eventID uint, in interface{})
		EncodeEvent(event WebsocketEventer, in interface{})
	}

	WebsocketEventer interface {
		EventID() uint
		UniqueID() []byte
		Decode(in interface{}) error
	}
)

func newWSSConnect(
	cid string,
	e func(id uint) (WebsocketServerHandler, bool),
	erw func(cid string, err error, msg string, args ...interface{}),
	wc *websocket.Conn,
	ctxs ...context.Context,
) *wssConn {
	ctx, cancel := context2.Combine(ctxs...)
	return &wssConn{
		status:  off,
		cid:     cid,
		ctx:     ctx,
		cancel:  cancel,
		onClose: make([]func(string), 0),
		onOpen:  make([]func(string), 0),
		sendC:   make(chan []byte, 128),
		erw:     erw,
		event:   e,
		conn:    wc,
	}
}

func (v *wssConn) ConnectID() string {
	return v.cid
}

func (v *wssConn) connect() *websocket.Conn {
	return v.conn
}

func (v *wssConn) cancelFunc() context.CancelFunc {
	return v.cancel
}

func (v *wssConn) done() <-chan struct{} {
	return v.ctx.Done()
}

func (v *wssConn) errLog(cid string, err error, msg string, args ...interface{}) {
	v.erw(cid, err, msg, args...)
}

func (v *wssConn) OnClose(cb func(cid string)) {
	lock(&v.mux, func() {
		v.onClose = append(v.onClose, cb)
	})
}

func (v *wssConn) OnOpen(cb func(cid string)) {
	lock(&v.mux, func() {
		v.onOpen = append(v.onOpen, cb)
	})
}

func (v *wssConn) Encode(eventID uint, in interface{}) {
	eventModel(func(ev *event) {
		ev.ID = eventID
		ev.Encode(in)
		b, err := json.Marshal(ev)
		if err != nil {
			v.errLog(v.cid, err, "[ws] encode message: %d", eventID)
			return
		}
		v.Write(b)
	})
}

func (v *wssConn) EncodeEvent(e WebsocketEventer, in interface{}) {
	eventModel(func(ev *event) {
		ev.ID = e.EventID()
		ev.UID = e.UniqueID()
		ev.Encode(in)
		b, err := json.Marshal(ev)
		if err != nil {
			v.errLog(v.cid, err, "[ws] encode message: %d", e.EventID())
			return
		}
		v.Write(b)
	})
}

func (v *wssConn) Write(b []byte) {
	if len(b) == 0 {
		return
	}

	select {
	case v.sendC <- b:
	default:
	}
}

func (v *wssConn) dataBus() <-chan []byte {
	return v.sendC
}

func (v *wssConn) dataHandler(b []byte) {
	eventModel(func(ev *event) {
		var (
			err error
			msg string
		)
		defer func() {
			if err != nil {
				v.errLog(v.cid, err, "[ws] "+msg)
			}
		}()
		if err = json.Unmarshal(b, ev); err != nil {
			msg = "decode message"
			return
		}
		call, ok := v.event(ev.EventID())
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

func (v *wssConn) Close() {
	if !atomic.CompareAndSwapInt64(&v.status, on, down) {
		return
	}
	v.errLog(v.ConnectID(), v.conn.Close(), "close connect")
}

func (v *wssConn) Run() {
	if !atomic.CompareAndSwapInt64(&v.status, off, on) {
		return
	}

	rwlock(&v.mux, func() {
		for _, fn := range v.onOpen {
			fn(v.ConnectID())
		}
	})

	setupPingPong(v.connect())
	go pumpWrite(v)
	pumpRead(v)

	rwlock(&v.mux, func() {
		for _, fn := range v.onClose {
			fn(v.ConnectID())
		}
	})
}
