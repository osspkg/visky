package main

import (
	"fmt"
	"os"

	"github.com/deweppro/go-sdk/console"
	"github.com/deweppro/goppy"
	"github.com/deweppro/goppy/plugins/web"
)

func main() {
	app := goppy.New()
	app.WithConfig("./config.yaml") // Reassigned via the `--config` argument when run via the console.
	app.Plugins(
		web.WithHTTP(),
	)
	app.Plugins()
	app.Command("env", func(s console.CommandSetter) {
		fmt.Println(os.Environ())
	})
	app.Run()
}
