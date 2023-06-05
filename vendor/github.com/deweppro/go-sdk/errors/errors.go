package errors

import (
	e "errors"
	"fmt"
)

type err struct {
	cause   error
	message string
	trace   string
}

func New(message string) error {
	return &err{message: message}
}

func (v *err) Error() string {
	switch true {
	case len(v.message) > 0 && v.cause != nil:
		return v.message + ": " + v.cause.Error() + v.trace
	case v.cause != nil:
		return v.cause.Error() + v.trace
	}
	return v.message + v.trace
}

func (v *err) Cause() error {
	return v.cause
}

func (v *err) Unwrap() error {
	return v.cause
}

func (v *err) WithTrace() {
	v.trace = tracing()
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func Trace(cause error, message string, args ...interface{}) error {
	v := Wrapf(cause, message, args...)
	if vv, ok := v.(*err); ok {
		vv.WithTrace()
		return vv
	}
	return v
}

func Wrapf(cause error, message string, args ...interface{}) error {
	if cause == nil {
		return nil
	}
	var err0 *err
	if len(args) == 0 {
		err0 = &err{
			cause:   cause,
			message: message,
		}
	} else {
		err0 = &err{
			cause:   cause,
			message: fmt.Sprintf(message, args...),
		}
	}
	return err0
}

func Wrap(msg ...error) error {
	if len(msg) == 0 {
		return nil
	}
	var err0 error
	for _, v := range msg {
		if v == nil {
			continue
		}
		if err0 == nil {
			err0 = &err{cause: v}
			continue
		}
		err0 = &err{
			cause:   v,
			message: err0.Error(),
		}
	}
	return err0
}

func Unwrap(err error) error {
	if v, ok := err.(interface {
		Unwrap() error
	}); ok {
		return v.Unwrap()
	}
	return nil
}

func Cause(err error) error {
	for err != nil {
		v, ok := err.(interface {
			Cause() error
		})
		if !ok {
			return err
		}
		err = v.Cause()
	}

	return nil
}

func Is(err, target error) bool {
	return e.Is(err, target)
}
