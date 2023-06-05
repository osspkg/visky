package web

//go:generate easyjson

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/deweppro/go-sdk/ioutil"
	"github.com/deweppro/go-sdk/log"
	"github.com/deweppro/go-sdk/webutil"
	"github.com/deweppro/go-static"
)

type (
	_ctx struct {
		w http.ResponseWriter
		r *http.Request
		l log.Logger
	}

	//Context request and response interface
	Context interface {
		URL() *url.URL
		Redirect(uri string)
		Param(key string) Param
		Header() Header
		Cookie() Cookie

		BindBytes(in *[]byte) error
		BindJSON(in interface{}) error
		BindXML(in interface{}) error
		Error(code int, err error)
		ErrorJSON(code int, err error, ctx ErrCtx)
		Bytes(code int, b []byte)
		String(code int, b string, args ...interface{})
		JSON(code int, in interface{})
		Stream(code int, in []byte, filename string)

		Context() context.Context
		Log() log.WriterContext
		Request() *http.Request
		Response() http.ResponseWriter
	}
)

func newContext(w http.ResponseWriter, r *http.Request, l log.Logger) Context {
	return &_ctx{
		w: w,
		r: r,
		l: l,
	}
}

/**********************************************************************************************************************/

func (v *_ctx) Request() *http.Request {
	return v.r
}

func (v *_ctx) Response() http.ResponseWriter {
	return v.w
}

/**********************************************************************************************************************/

type (
	//Param interface for typing a parameter from a URL
	Param interface {
		String() (string, error)
		Int() (int64, error)
		Float() (float64, error)
	}
	_param struct {
		val string
		err error
	}
)

// String getting the parameter as a string
func (v _param) String() (string, error) { return v.val, v.err }

// Int getting the parameter as a int64
func (v _param) Int() (int64, error) {
	if v.err != nil {
		return 0, v.err
	}
	return strconv.ParseInt(v.val, 10, 64)
}

// Float getting the parameter as a float64
func (v _param) Float() (float64, error) {
	if v.err != nil {
		return 0.0, v.err
	}
	return strconv.ParseFloat(v.val, 64)
}

// Param getting a parameter from URL by key
func (v *_ctx) Param(key string) Param {
	val, err := webutil.ParamString(v.r, key)
	return _param{
		val: val,
		err: err,
	}
}

/**********************************************************************************************************************/

// Log log entry interface
func (v *_ctx) Log() log.WriterContext {
	return v.l
}

/**********************************************************************************************************************/

type (
	Header interface {
		Get(key string) string
		Set(key, value string)
		Del(key string)
		Val(key string) string
		Copy()
	}

	_header struct {
		r http.Header
		w http.Header
	}
)

func (v *_header) Get(key string) string {
	return v.r.Get(key)
}

func (v *_header) Set(key, value string) {
	v.w.Set(key, value)
}

func (v *_header) Del(key string) {
	v.w.Del(key)
}

func (v *_header) Val(key string) string {
	return v.w.Get(key)
}

func (v *_header) Copy() {
	for key := range v.r {
		v.w.Set(key, v.r.Get(key))
	}
}

func (v *_ctx) Header() Header {
	return &_header{
		r: v.r.Header,
		w: v.w.Header(),
	}
}

/**********************************************************************************************************************/

type (
	Cookie interface {
		Get(key string) *http.Cookie
		Set(value *http.Cookie)
	}

	_cookie struct {
		r *http.Request
		w http.ResponseWriter
	}
)

// Get getting cookies from a key request
func (v *_cookie) Get(key string) *http.Cookie {
	c, _ := v.r.Cookie(key) //nolint: errcheck
	return c
}

// Set setting cookies in response
func (v *_cookie) Set(value *http.Cookie) {
	http.SetCookie(v.w, value)
}

func (v *_ctx) Cookie() Cookie {
	return &_cookie{
		r: v.r,
		w: v.w,
	}
}

/**********************************************************************************************************************/

func (v *_ctx) BindBytes(in *[]byte) error {
	b, err := ioutil.ReadAll(v.r.Body)
	if err != nil {
		return err
	}
	*in = append(*in, b...)
	return nil
}

func (v *_ctx) BindJSON(in interface{}) error {
	return webutil.JSONDecode(v.r, in)
}

func (v *_ctx) BindXML(in interface{}) error {
	return webutil.XMLDecode(v.r, in)
}

/**********************************************************************************************************************/

//easyjson:json
type (
	errMessage struct {
		Message string `json:"msg"`
		Ctx     ErrCtx `json:"ctx,omitempty"`
	}

	ErrCtx map[string]interface{}
)

func (v *_ctx) Error(code int, err error) {
	if err == nil {
		err = fmt.Errorf("unknown error")
	}
	http.Error(v.w, err.Error(), code)
}

func (v *_ctx) ErrorJSON(code int, err error, ctx ErrCtx) {
	if err == nil {
		err = fmt.Errorf("unknown error")
	}
	model := errMessage{
		Message: err.Error(),
		Ctx:     ctx,
	}
	b, _ := json.Marshal(&model) //nolint: errcheck
	v.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	v.w.WriteHeader(code)
	if _, err = v.w.Write(b); err != nil {
		v.l.WithFields(log.Fields{
			"err": err.Error(),
		}).Errorf("ErrorJSON response")
	}
}

func (v *_ctx) Bytes(code int, b []byte) {
	v.w.WriteHeader(code)
	if _, err := v.w.Write(b); err != nil {
		v.l.WithFields(log.Fields{
			"err": err.Error(),
		}).Errorf("Bytes response")
	}
}

// String recording the response in string format
func (v *_ctx) String(code int, b string, args ...interface{}) {
	v.w.WriteHeader(code)
	if _, err := fmt.Fprintf(v.w, b, args...); err != nil {
		v.l.WithFields(log.Fields{
			"err": err.Error(),
		}).Errorf("String response")
	}
}

// JSON recording the response in json format
func (v *_ctx) JSON(code int, in interface{}) {
	b, err := json.Marshal(in)
	if err != nil {
		v.ErrorJSON(http.StatusInternalServerError, err, nil)
		return
	}
	v.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	v.w.WriteHeader(code)
	if _, err = v.w.Write(b); err != nil {
		v.l.WithFields(log.Fields{
			"err": err.Error(),
		}).Errorf("JSON response")
	}
}

// Stream sending raw data in response with the definition of the content type by the file name
func (v *_ctx) Stream(code int, in []byte, filename string) {
	contentType := static.DetectContentType(filename, in)
	v.w.Header().Set("Content-Type", contentType)
	v.w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	v.w.WriteHeader(code)
	if _, err := v.w.Write(in); err != nil {
		v.l.WithFields(log.Fields{
			"err": err.Error(),
		}).Errorf("Stream response")
	}
}

/**********************************************************************************************************************/

// Context provider the request context
func (v *_ctx) Context() context.Context {
	return v.r.Context()
}

/**********************************************************************************************************************/

// URL getting a URL from a request
func (v *_ctx) URL() *url.URL {
	uri := v.r.URL
	uri.Host = v.r.Host
	return uri
}

/**********************************************************************************************************************/

// Redirect redirecting to another URL
func (v *_ctx) Redirect(uri string) {
	http.Redirect(v.w, v.r, uri, http.StatusMovedPermanently)
}
