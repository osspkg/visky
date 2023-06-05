package webutil

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var uriParamRex = regexp.MustCompile(`\{([a-z0-9]+)\:?([^{}]*)\}`)

type paramMatch struct {
	incr    int
	keys    map[string]string
	links   map[string]string
	pattern string
	rex     *regexp.Regexp
}

func newParamMatch() *paramMatch {
	return &paramMatch{
		incr:    1,
		pattern: "",
		keys:    make(map[string]string),
		links:   make(map[string]string),
	}
}

func (v *paramMatch) Add(vv string) error {
	result := vv

	patterns := uriParamRex.FindAllString(vv, -1)
	for _, pattern := range patterns {
		res := uriParamRex.FindAllStringSubmatch(pattern, 1)[0]

		key := fmt.Sprintf("k%d", v.incr)
		rex := ".+"
		if len(res) == 3 && len(res[2]) > 0 {
			rex = res[2]
		}
		result = strings.Replace(result, res[0], fmt.Sprintf("(?P<%s>%s)", key, rex), 1)

		v.links[key] = vv
		v.keys[key] = res[1]
		v.incr++
	}

	result = "^" + result + "$"

	if _, err := regexp.Compile(result); err != nil {
		return fmt.Errorf("regex compilation error for `%s`: %w", vv, err)
	}

	if len(v.pattern) != 0 {
		v.pattern += "|"
	}
	v.pattern += result
	v.rex = regexp.MustCompile(v.pattern)
	return nil
}

func (v *paramMatch) Match(vv string, vr uriParamData) (string, bool) {
	if v.rex == nil {
		return "", false
	}

	matches := v.rex.FindStringSubmatch(vv)
	if len(matches) == 0 {
		return "", false
	}

	link := ""
	for indx, name := range v.rex.SubexpNames() {
		val := matches[indx]
		if len(val) == 0 {
			continue
		}
		if l, ok := v.links[name]; ok {
			link = l
		}
		if key, ok := v.keys[name]; ok {
			vr[key] = val
		}
	}

	return link, true
}

func hasParamMatch(vv string) bool {
	return uriParamRex.MatchString(vv)
}

/**********************************************************************************************************************/

type (
	uriParamKey  string
	uriParamData map[string]string
)

func ParamString(r *http.Request, key string) (string, error) {
	if v := r.Context().Value(uriParamKey(key)); v != nil {
		return v.(string), nil
	}
	return "", errFailContextKey
}

func ParamInt(r *http.Request, key string) (int64, error) {
	v, err := ParamString(r, key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(v, 10, 64)
}

func ParamFloat(r *http.Request, key string) (float64, error) {
	v, err := ParamString(r, key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(v, 64)
}
