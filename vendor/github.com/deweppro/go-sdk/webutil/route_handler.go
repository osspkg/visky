package webutil

import (
	"net/http"
	"strings"
)

const anyPath = "#"

type ctrlHandler struct {
	list        map[string]*ctrlHandler
	methods     map[string]func(http.ResponseWriter, *http.Request)
	matcher     *paramMatch
	middlewares []func(func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request)
	notFound    func(http.ResponseWriter, *http.Request)
}

func newCtrlHandler() *ctrlHandler {
	return &ctrlHandler{
		list:        make(map[string]*ctrlHandler),
		methods:     make(map[string]func(http.ResponseWriter, *http.Request)),
		matcher:     newParamMatch(),
		middlewares: make([]func(func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request), 0),
	}
}

func (v *ctrlHandler) append(path string) *ctrlHandler {
	if uh, ok := v.list[path]; ok {
		return uh
	}
	uh := newCtrlHandler()
	v.list[path] = uh
	return uh
}

func (v *ctrlHandler) next(path string, vars uriParamData) (*ctrlHandler, bool) {
	if uh, ok := v.list[path]; ok {
		return uh, false
	}
	if uri, ok := v.matcher.Match(path, vars); ok {
		if uh, ok1 := v.list[uri]; ok1 {
			return uh, false
		}
	}
	if uh, ok := v.list[anyPath]; ok {
		return uh, true
	}
	return nil, false
}

// Route add new route
func (v *ctrlHandler) Route(path string, ctrl func(http.ResponseWriter, *http.Request), methods []string) {
	uh := v
	uris := urlSplit(path)
	for _, uri := range uris {
		if hasParamMatch(uri) {
			if err := uh.matcher.Add(uri); err != nil {
				panic(err)
			}
		}
		uh = uh.append(uri)
	}
	for _, m := range methods {
		uh.methods[strings.ToUpper(m)] = ctrl
	}
}

// Middlewares add middleware to route
func (v *ctrlHandler) Middlewares(
	path string, middlewares ...func(func(http.ResponseWriter, *http.Request),
	) func(http.ResponseWriter, *http.Request)) {
	uh := v
	uris := urlSplit(path)
	for _, uri := range uris {
		uh = uh.append(uri)
	}
	uh.middlewares = append(uh.middlewares, middlewares...)
}

func (v *ctrlHandler) NoFoundHandler(call func(http.ResponseWriter, *http.Request)) {
	v.notFound = call
}

// Match find route in tree
func (v *ctrlHandler) Match(path string, method string) (
	int, func(http.ResponseWriter, *http.Request), uriParamData, []func(func(http.ResponseWriter, *http.Request),
	) func(http.ResponseWriter, *http.Request)) {
	uh := v
	uris := urlSplit(path)
	midd := append(make([]func(func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request),
		0, len(uh.middlewares)), uh.middlewares...)
	vr := uriParamData{}
	var isBreak bool
	for _, uri := range uris {
		if uh, isBreak = uh.next(uri, vr); uh != nil {
			midd = append(midd, uh.middlewares...)
			if isBreak {
				break
			}
			continue
		}
		if v.notFound != nil {
			return http.StatusOK, v.notFound, nil, midd
		}
		return http.StatusNotFound, nil, nil, v.middlewares
	}
	if ctrl, ok := uh.methods[method]; ok {
		return http.StatusOK, ctrl, vr, midd
	}
	if v.notFound != nil {
		return http.StatusOK, v.notFound, nil, midd
	}
	if len(uh.methods) == 0 {
		return http.StatusNotFound, nil, nil, v.middlewares
	}
	return http.StatusMethodNotAllowed, nil, nil, v.middlewares
}
