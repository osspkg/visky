package images

import (
	"github.com/deweppro/goppy/plugins"
)

var Plugin = plugins.Plugin{
	Inject: func() *Images {
		return New()
	},
}
