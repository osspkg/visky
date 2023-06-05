package static

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// FromBase64TarGZ ...
func (c *Cache) FromBase64TarGZ(v string) error {
	b64, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return err
	}
	return c.FromTarGZArchive(bytes.NewBuffer(b64))
}

// FromTarGZArchive ...
func (c *Cache) FromTarGZArchive(r io.Reader) error {
	gzf, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	return c.FromTarArchive(gzf)
}

// FromTarArchive ...
func (c *Cache) FromTarArchive(r io.Reader) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		c.Set(hdr.Name, b)
	}
	return nil
}

// ToTarGZArchive ...
func (c *Cache) ToTarGZArchive(w io.Writer) error {
	gw := gzip.NewWriter(w)
	err := c.ToTarArchive(gw)
	if err != nil {
		return err
	}
	if err = gw.Close(); err != nil {
		return err
	}
	return nil
}

// ToTarArchive ...
func (c *Cache) ToTarArchive(w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close() //nolint:errcheck

	c.mux.RLock()
	defer c.mux.RUnlock()

	for _, name := range c.List() {
		v, ok := c.files[name]
		if !ok {
			return fmt.Errorf("file not found: %s", name)
		}
		hdr := &tar.Header{
			Name: name,
			Mode: int64(os.ModePerm),
			Size: int64(len(v)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(v); err != nil {
			return err
		}
	}
	return nil
}

// ToBase64TarGZ ...
func (c *Cache) ToBase64TarGZ() (string, error) {
	buf := &bytes.Buffer{}
	if err := c.ToTarGZArchive(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
