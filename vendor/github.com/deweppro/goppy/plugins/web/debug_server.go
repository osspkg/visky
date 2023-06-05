package web

import (
	"github.com/deweppro/go-sdk/log"
	"github.com/deweppro/go-sdk/webutil"
	"github.com/deweppro/goppy/plugins"
)

// ConfigDebug config to initialize HTTP debug service
type ConfigDebug struct {
	Config webutil.ConfigHttp `yaml:"debug"`
}

func (v *ConfigDebug) Default() {
	v.Config = webutil.ConfigHttp{Addr: "127.0.0.1:12000"}
}

// WithHTTPDebug debug service over HTTP protocol with pprof enabled
func WithHTTPDebug() plugins.Plugin {
	return plugins.Plugin{
		Config: &ConfigDebug{},
		Inject: func(c *ConfigDebug, l log.Logger) *webutil.ServerDebug {
			return webutil.NewServerDebug(c.Config, l)
		},
	}
}
