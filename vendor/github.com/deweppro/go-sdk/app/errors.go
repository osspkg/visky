package app

import "github.com/deweppro/go-sdk/errors"

var (
	errDepBuilderNotRunning = errors.New("dependencies builder is not running")
	errDepNotRunning        = errors.New("dependencies are not running yet")
	errServiceUnknown       = errors.New("unknown service")
	errBadFileFormat        = errors.New("is not a supported file format")
)
