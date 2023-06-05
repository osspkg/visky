package static

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	//ErrUndefinedFormat error
	ErrUndefinedFormat = errors.New("undefined format")
)

func (c *Cache) fileCreate(filename string, call func(r io.Writer) error) error {
	v, err := os.Create(filename)
	if err != nil {
		return err
	}
	if err = call(v); err != nil {
		return err
	}
	if err = v.Close(); err != nil {
		return err
	}
	return nil
}

func (c *Cache) fileOpen(filename string, call func(r io.Reader) error) error {
	v, err := os.Open(filename)
	if err != nil {
		return err
	}
	if err = call(v); err != nil {
		return err
	}
	if err = v.Close(); err != nil {
		return err
	}
	return nil
}

// FromFile ...
func (c *Cache) FromFile(filename string) error {
	switch true {
	case strings.HasSuffix(filename, ".tar.gz"):
		return c.fileOpen(filename, func(v io.Reader) error {
			return c.FromTarGZArchive(v)
		})

	case strings.HasSuffix(filename, ".tar"):
		return c.fileOpen(filename, func(v io.Reader) error {
			return c.FromTarArchive(v)
		})

	default:
		return ErrUndefinedFormat
	}
}

// FromDir ...
func (c *Cache) FromDir(dir string) error {
	if v, err := filepath.Abs(dir); err == nil {
		dir = v
	} else {
		return err
	}
	return filepath.Walk(dir, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, "_static.go") {
			return nil
		}
		if b, err := ioutil.ReadFile(path); err != nil {
			return err
		} else {
			path = strings.TrimPrefix(path, dir)
			c.Set(path, b)
		}
		return nil
	})
}

// ToFile ...
func (c *Cache) ToFile(filename string) error {
	switch true {
	case strings.HasSuffix(filename, ".tar.gz"):
		return c.fileCreate(filename, func(w io.Writer) error {
			return c.ToTarGZArchive(w)
		})

	case strings.HasSuffix(filename, ".tar"):
		return c.fileCreate(filename, func(w io.Writer) error {
			return c.ToTarArchive(w)
		})

	default:
		return ErrUndefinedFormat
	}
}
