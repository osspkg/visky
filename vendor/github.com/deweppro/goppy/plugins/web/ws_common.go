package web

import (
	"context"
	"net/http"
	"time"

	"github.com/deweppro/go-sdk/errors"
	"github.com/gorilla/websocket"
)

const (
	on   = 1
	off  = 0
	down = 2
)

var (
	errServAlreadyRunning = errors.New("server already running")
	errServAlreadyStopped = errors.New("server already stopped")
	errOneOpenConnect     = errors.New("connection can be started once")

	wsu = websocket.Upgrader{
		EnableCompression: true,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}
)

/**********************************************************************************************************************/

const (
	pongWait   = 60 * time.Second
	pingPeriod = pongWait / 3
)

func setupPingPong(c *websocket.Conn) {
	c.SetPingHandler(func(_ string) error {
		return errors.Wrap(
			c.SetReadDeadline(time.Now().Add(pongWait)),
			//v.conn.SetWriteDeadline(time.Now().Add(pongWait)),
		)
	})
	c.SetPongHandler(func(_ string) error {
		return errors.Wrap(
			c.SetReadDeadline(time.Now().Add(pongWait)),
			//v.conn.SetWriteDeadline(time.Now().Add(pongWait)),
		)
	})
}

/**********************************************************************************************************************/

type processor interface {
	ConnectID() string
	dataHandler(b []byte)
	dataBus() <-chan []byte
	connect() *websocket.Conn
	cancelFunc() context.CancelFunc
	done() <-chan struct{}
	errLog(cid string, err error, msg string, args ...interface{})
	Close()
}

func pumpRead(p processor) {
	defer p.cancelFunc()
	for {
		_, message, err := p.connect().ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, 1000, 1001, 1005) {
				p.errLog(p.ConnectID(), err, "[ws] read message")
			}
			return
		}
		go p.dataHandler(message)
	}
}

func pumpWrite(p processor) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		p.errLog(p.ConnectID(), p.connect().Close(), "close connect")
	}()
	for {
		select {
		case <-p.done():
			err := p.connect().WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Bye bye!"))
			if err != nil && !errors.Is(err, websocket.ErrCloseSent) {
				p.errLog(p.ConnectID(), err, "[ws] send close")
			}
			return
		case m := <-p.dataBus():
			if err := p.connect().WriteMessage(websocket.TextMessage, m); err != nil {
				p.errLog(p.ConnectID(), err, "[ws] send message")
				return
			}
		case <-ticker.C:
			if err := p.connect().WriteMessage(websocket.PingMessage, nil); err != nil {
				p.errLog(p.ConnectID(), err, "[ws] send ping")
				return
			}
		}
	}
}
