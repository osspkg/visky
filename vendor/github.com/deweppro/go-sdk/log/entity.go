package log

import (
	"fmt"
	"os"
	"reflect"
)

type entity struct {
	log Sender
	ctx Fields
}

func newEntity(log Sender) *entity {
	return &entity{
		log: log,
		ctx: Fields{},
	}
}

func (e *entity) Reset() {
	e.ctx = Fields{}
}

func (e *entity) WithError(key string, err error) Writer {
	if err != nil {
		e.ctx[key] = err.Error()
	} else {
		e.ctx[key] = nil
	}
	return e
}

func (e *entity) WithField(key string, value interface{}) Writer {
	ref := reflect.TypeOf(value)
	if ref != nil {
		switch ref.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Ptr, reflect.Struct:
			e.ctx[key] = fmt.Sprintf("unsupported field value: %#v", value)
			return e
		}
	}
	e.ctx[key] = value
	return e
}

func (e *entity) WithFields(fields Fields) Writer {
	for key, value := range fields {
		ref := reflect.TypeOf(value)
		if ref != nil {
			switch ref.Kind() {
			case reflect.Chan, reflect.Func, reflect.Interface, reflect.Ptr, reflect.Struct:
				e.ctx[key] = fmt.Sprintf("unsupported field value: %#v", value)
				continue
			}
		}
		e.ctx[key] = value
	}
	return e
}

func (e *entity) prepareMessage(format string, args ...interface{}) func(v *message) {
	return func(v *message) {
		v.Message = fmt.Sprintf(format, args...)
		for key, value := range e.ctx {
			v.Ctx[key] = value
		}
		e.log.PutEntity(e)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Infof info message
func (e *entity) Infof(format string, args ...interface{}) {
	e.log.SendMessage(LevelInfo, e.prepareMessage(format, args...))
}

// Warnf warning message
func (e *entity) Warnf(format string, args ...interface{}) {
	e.log.SendMessage(LevelWarn, e.prepareMessage(format, args...))
}

// Errorf error message
func (e *entity) Errorf(format string, args ...interface{}) {
	e.log.SendMessage(LevelError, e.prepareMessage(format, args...))
}

// Debugf debug message
func (e *entity) Debugf(format string, args ...interface{}) {
	e.log.SendMessage(LevelDebug, e.prepareMessage(format, args...))
}

// Fatalf fatal message and exit
func (e *entity) Fatalf(format string, args ...interface{}) {
	e.log.SendMessage(levelFatal, e.prepareMessage(format, args...))
	e.log.Close()
	os.Exit(1)
}
