package web

import (
	"encoding/json"
	"sync"
)

var (
	poolWSEvent = sync.Pool{New: func() interface{} { return &event{} }}
)

//easyjson:json
type event struct {
	ID      uint            `json:"e"`
	Data    json.RawMessage `json:"d"`
	Err     *string         `json:"err,omitempty"`
	UID     json.RawMessage `json:"u,omitempty"`
	Updated bool            `json:"-"`
}

func (v *event) EventID() uint {
	return v.ID
}

func (v *event) UniqueID() []byte {
	if v.UID == nil {
		return nil
	}
	result := make([]byte, 0, len(v.UID))
	return append(result, v.UID...)
}

func (v *event) Decode(in interface{}) error {
	return json.Unmarshal(v.Data, in)
}

func (v *event) Encode(in interface{}) {
	b, err := json.Marshal(in)
	if err != nil {
		v.Error(err)
		return
	}
	v.Body(b)
}

func (v *event) Reset() *event {
	v.ID, v.Err, v.UID, v.Data, v.Updated = 0, nil, nil, v.Data[:0], false
	return v
}

func (v *event) Error(e error) {
	if e == nil {
		return
	}
	err := e.Error()
	v.Err, v.Data, v.Updated = &err, v.Data[:0], true
}

func (v *event) Body(b []byte) {
	v.Err, v.Data, v.Updated = nil, append(v.Data[:0], b...), true
}

func eventModel(call func(ev *event)) {
	m, ok := poolWSEvent.Get().(*event)
	if !ok {
		m = &event{}
	}
	call(m)
	poolWSEvent.Put(m.Reset())
}
