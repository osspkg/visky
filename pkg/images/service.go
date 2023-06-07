package images

import (
	"crypto/sha1"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/deweppro/go-sdk/errors"
	"github.com/deweppro/go-sdk/log"
	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

var (
	ExtNotSupported = errors.New("ext is not supported")
)

type (
	Images struct {
		folder string
		cache  map[string]ImageInfo
		mux    sync.Mutex
	}

	ImageInfo struct {
		Hash   string
		Origin string
		Scale  string
		Thumb  string
	}
)

var decoders = map[string]func(r io.Reader) (image.Image, error){
	".jpg":  jpeg.Decode,
	".jpeg": jpeg.Decode,
	".png":  png.Decode,
	".webp": webp.Decode,
	".bmp":  bmp.Decode,
	".tiff": tiff.Decode,
}

func New() *Images {
	return &Images{
		folder: "",
		cache:  make(map[string]ImageInfo, 100),
	}
}

func (v *Images) SetFolder(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create image folder: %w", err)
	}
	v.folder = dir
	return nil
}

func (v *Images) Build(filename string, scale, thumb int) (*ImageInfo, error) {
	var err error
	img := &ImageInfo{}

	if img.Hash, err = v.getHash(filename); err != nil {
		return nil, err
	}

	v.mux.Lock()
	i, ok := v.cache[img.Hash]
	v.mux.Unlock()
	if ok {
		return &i, nil
	}

	if img.Origin, err = v.resize(filename, img.Hash+".orig", 0); err != nil {
		return nil, err
	}
	if img.Scale, err = v.resize(filename, img.Hash+".scale", scale); err != nil {
		return nil, err
	}
	if img.Thumb, err = v.resize(filename, img.Hash+".thumb", thumb); err != nil {
		return nil, err
	}

	v.mux.Lock()
	v.cache[img.Hash] = *img
	v.mux.Unlock()

	return img, nil
}

func (v *Images) resize(filename, suffix string, width int) (string, error) {
	src, name, err := v.readFile(filename)
	if err != nil {
		return "", err
	}
	x, y := v.scaleFactor(src.Bounds().Max.X, src.Bounds().Max.Y, width)
	dst := image.NewNRGBA(image.Rect(0, 0, x, y))
	draw.CatmullRom.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)
	newFilename := fmt.Sprintf("%s-%s.png", name, suffix)
	return newFilename, v.writeFile(v.folder+"/"+newFilename, dst)
}

func (v *Images) scaleFactor(oW, oH, width int) (int, int) {
	if width == 0 {
		return oW, oH
	}
	oWF, oHF := float64(oW), float64(oH)
	nWidth := float64(width)
	scale := oWF / nWidth
	return int(oWF / scale), int(oHF / scale)
}

func (v *Images) writeFile(filename string, img image.Image) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("write image `%s`: %w", filename, err)
	}
	defer func() {
		if err0 := file.Close(); err0 != nil {
			log.WithFields(log.Fields{
				"err":  err0.Error(),
				"file": filename,
			}).Errorf("Close image file")
		}
	}()
	if err = png.Encode(file, img); err != nil {
		return fmt.Errorf("encode image `%s`: %w", filename, err)
	}
	return nil
}

func (v *Images) readFile(filename string) (image.Image, string, error) {
	ext := filepath.Ext(filename)
	dec, ok := decoders[ext]
	if !ok {
		return nil, "", ExtNotSupported
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, "", fmt.Errorf("read image `%s`: %w", filename, err)
	}
	defer func() {
		if err0 := file.Close(); err0 != nil {
			log.WithFields(log.Fields{
				"err":  err0.Error(),
				"file": filename,
			}).Errorf("Close image file")
		}
	}()

	img, err := dec(file)
	if err != nil {
		return nil, "", fmt.Errorf("decode image `%s`: %w", filename, err)
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, "", fmt.Errorf("info image `%s`: %w", filename, err)
	}
	return img, strings.Replace(fi.Name(), ext, "", 1), nil
}

func (v *Images) getHash(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("read image `%s`: %w", filename, err)
	}
	defer func() {
		if err0 := file.Close(); err0 != nil {
			log.WithFields(log.Fields{
				"err":  err0.Error(),
				"file": filename,
			}).Errorf("Close image file")
		}
	}()

	h := sha1.New()
	if _, err = io.Copy(h, file); err != nil {
		return "", fmt.Errorf("calc hash image `%s`: %w", filename, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
