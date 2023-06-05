package log

import "sync"

//go:generate easyjson

var poolMessage = sync.Pool{
	New: func() interface{} {
		return newMessage()
	},
}

//easyjson:json
type message struct {
	Time    int64                  `json:"time"`
	Level   string                 `json:"lvl"`
	Message string                 `json:"msg"`
	Ctx     map[string]interface{} `json:"ctx,omitempty"`
}

func newMessage() *message {
	return &message{
		Ctx: make(map[string]interface{}),
	}
}

func (v *message) Reset() {
	v.Time = 0
	v.Level = ""
	v.Message = ""
	for s := range v.Ctx {
		delete(v.Ctx, s)
	}
}
