package images

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"png.adpollak.net/internal/chunk"
)

// CreateImage takes in deflated IDAT pixel data, and IHDR chunk data to
// recreate a PNG image. It then returns said image.
func CreateImage(pixels []byte, ihdr chunk.IHDR) (image.Image, error) {
	// Switch on the 5 color types as specified in the PNG specification.
	var img image.Image
	width := ihdr.Width
	height := ihdr.Height
	switch ihdr.ColorType {
	case 0:
		img = handleGreyscale(pixels, int(width), int(height))
		log.Println("ColorType: Greyscale")
		// TODO: handle Greyscale;
	case 2:
		log.Println("ColorType : Truecolor")
	case 3:
		log.Println("ColorType: Indexed-color")
	case 4:
		log.Println("ColorType: Greyscale with alpha")
	case 6:
		log.Println("ColorType: Truecolor with alpha")
	default:
		return nil, fmt.Errorf("invalid ColorType: %v", ihdr.ColorType)
	}
	// TODO: create the image, and return it based on the handler

	return img, nil
}

func handleGreyscale(pixels []byte, width, height int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))
	// size each pixel = num bits per pixel
	// img = rectangular pixel array
	// pixels w/in a scanline packed into a sequence of bytes
	// greyscale: each pixel is one sample
	// TODO: Need to traverse the pixels, and populate our grayscale image with it's information

	scanline := width
	for r := 0; r < height; r++ {
		for c := 0; c < width; c++ {
			// Remember, our pixels are stored IN-MEMORY as a 1D ARRAY of pixel values
			// Thus, we need an OFFSET to determine WHERE to access a specific element.
			// Traverse each pixel group; l -> r, t -> b
			offset := r*scanline + c
			if offset >= len(pixels) {
				break
			}
			greyValue := pixels[offset]
			img.SetGray(r, c, color.Gray{Y: greyValue})
		}
	}
	return img
}
