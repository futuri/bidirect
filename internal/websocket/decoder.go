package websocket

import (
	"bytes"
	"image"
	"image/color"
	"math"

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

// CreateBiDirectLogo creates a simple "BD" logo with arrows
func CreateBiDirectLogo(size int) []byte {
	bgra := make([]byte, size*size*4)
	center := float64(size) / 2
	radius := float64(size) * 0.4

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			idx := (y*size + x) * 4
			fx, fy := float64(x)-center, float64(y)-center
			dist := math.Sqrt(fx*fx + fy*fy)

			// Circle with gradient
			if dist < radius {
				// Inside circle - gradient blue
				t := dist / radius
				bgra[idx+0] = uint8(200 - t*100) // B
				bgra[idx+1] = uint8(100 - t*50)  // G
				bgra[idx+2] = uint8(50 + t*50)   // R
				bgra[idx+3] = 255                // A
			} else if dist < radius+3 {
				// Border - white
				bgra[idx+0] = 255
				bgra[idx+1] = 255
				bgra[idx+2] = 255
				bgra[idx+3] = 255
			} else {
				// Outside - transparent
				bgra[idx+3] = 0
			}
		}
	}

	// Draw arrows (bidirectional symbol)
	drawArrow(bgra, size, int(center), int(center), -1, 0, int(radius*0.6)) // Left arrow
	drawArrow(bgra, size, int(center), int(center), 1, 0, int(radius*0.6))  // Right arrow

	return bgra
}

func drawArrow(bgra []byte, size, cx, cy, dx, dy, length int) {
	// Arrow line
	for i := 0; i < length; i++ {
		x := cx + dx*i
		y := cy + dy*i
		if x >= 0 && x < size && y >= 0 && y < size {
			for py := -2; py <= 2; py++ {
				for px := -2; px <= 2; px++ {
					nx, ny := x+px, y+py
					if nx >= 0 && nx < size && ny >= 0 && ny < size {
						idx := (ny*size + nx) * 4
						bgra[idx+0] = 255
						bgra[idx+1] = 255
						bgra[idx+2] = 255
						bgra[idx+3] = 255
					}
				}
			}
		}
	}

	// Arrow head
	tipX := cx + dx*length
	tipY := cy + dy*length
	for i := 0; i < 15; i++ {
		for j := -i; j <= i; j++ {
			x := tipX - dx*i
			y := tipY + j
			if dx == 0 {
				x = tipX + j
				y = tipY - dy*i
			}
			if x >= 0 && x < size && y >= 0 && y < size {
				idx := (y*size + x) * 4
				bgra[idx+0] = 255
				bgra[idx+1] = 255
				bgra[idx+2] = 255
				bgra[idx+3] = 255
			}
		}
	}
}
