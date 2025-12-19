package websocket

import (
	"bytes"
	"image"
	"image/color"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/webp"
)

func DecodeImageToBGRA(data []byte) ([]byte, int, int, error) {
	var img image.Image
	var err error

	img, err = webp.Decode(bytes.NewReader(data))
	if err != nil {
		img, _, err = image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, 0, 0, err
		}
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	bgra := make([]byte, width*height*4)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			idx := (y*width + x) * 4
			bgra[idx+0] = uint8(b >> 8)
			bgra[idx+1] = uint8(g >> 8)
			bgra[idx+2] = uint8(r >> 8)
			bgra[idx+3] = uint8(a >> 8)
		}
	}

	return bgra, width, height, nil
}

func CreateBlankFrame(width, height int, col color.NRGBA) []byte {
	bgra := make([]byte, width*height*4)
	for i := 0; i < width*height; i++ {
		idx := i * 4
		bgra[idx+0] = col.B
		bgra[idx+1] = col.G
		bgra[idx+2] = col.R
		bgra[idx+3] = col.A
	}
	return bgra
}
