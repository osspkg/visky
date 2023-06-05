package web

import (
	"github.com/deweppro/go-sdk/log"
	"github.com/deweppro/go-sdk/webutil"
	"github.com/deweppro/goppy/plugins"
)

// ConfigHttp config to initialize HTTP service
type ConfigHttp struct {
	Config map[string]webutil.ConfigHttp `yaml:"http"`
}

func (v *ConfigHttp) Default() {
	if v.Config == nil {
		v.Config = map[string]webutil.ConfigHttp{
			"main": {Addr: "127.0.0.1:8080"},
		}
	}
}

// WithHTTP launch of HTTP service with default Router
func WithHTTP() plugins.Plugin {
	return plugins.Plugin{
		Config: &ConfigHttp{},
		Inject: func(conf *ConfigHttp, l log.Logger) (*routeProvider, RouterPool) {
			rp := newRouteProvider(conf.Config, l)
			return rp, rp
		},
	}
}
