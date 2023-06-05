package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/deweppro/go-sdk/errors"
	"gopkg.in/yaml.v3"
)

// Sources model
type Sources string

// Decode unmarshal file to model
func (v Sources) Decode(configs ...interface{}) error {
	data, err := os.ReadFile(string(v))
	if err != nil {
		return err
	}
	ext := filepath.Ext(string(v))
	switch ext {
	case ".yml", ".yaml":
		return v.unmarshal("yaml unmarshal", data, yaml.Unmarshal, configs...)
	case ".json":
		return v.unmarshal("json unmarshal", data, json.Unmarshal, configs...)
	}
	return errBadFileFormat
}

func (v Sources) unmarshal(
	title string, data []byte, call func([]byte, interface{},
	) error, configs ...interface{}) error {
	for _, conf := range configs {
		if err := call(data, conf); err != nil {
			return errors.Wrapf(err, title)
		}
	}
	return nil
}
