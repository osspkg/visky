package pool

import (
	"bytes"
	"io"
	"sync"
)

type Writer interface {
	io.StringWriter
	io.Writer
}

var bytesPool = sync.Pool{
	New: func() interface{} { return bytes.NewBuffer([]byte{}) },
}

func ResolveBytes(call func(w Writer)) []byte {
	buf, ok := bytesPool.Get().(*bytes.Buffer)
	if !ok {
		buf = bytes.NewBuffer([]byte{})
	}
	call(buf)
	b := append([]byte{}, buf.Bytes()...)
	buf.Reset()
	bytesPool.Put(buf)
	return b
}
