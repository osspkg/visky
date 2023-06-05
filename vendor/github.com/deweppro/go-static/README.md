# go-static

[![Coverage Status](https://coveralls.io/repos/github/deweppro/go-static/badge.svg?branch=master)](https://coveralls.io/github/deweppro/go-static?branch=master)
[![Release](https://img.shields.io/github/release/deweppro/go-static.svg?style=flat-square)](https://github.com/deweppro/go-static/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/deweppro/go-static)](https://goreportcard.com/report/github.com/deweppro/go-static)
[![CI](https://github.com/deweppro/go-static/actions/workflows/ci.yml/badge.svg)](https://github.com/deweppro/go-static/actions/workflows/ci.yml)

_Library for embedding static files inside an application_

# Install as tool

```bash
go install github.com/deweppro/go-static/cmd/static@latest
```

# Packaging

```go
//go:generate static <DIR> <VAR>
```

* DIR - Path to the static folder
* VAR - A variable containing `static.Reader` interface

## Example go code

```go
package example

import (
	"fmt"

	"github.com/deweppro/go-static"
)

//go:generate static ./.. ui

var ui static.Reader

func run() {
	fmt.Println(ui.List())
}
```
## License

BSD-3-Clause License. See the LICENSE file for details.