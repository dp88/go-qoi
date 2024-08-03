package qoi

import (
	"errors"
	"image"
	"image/color"
	"io"
)

const (
	// Chunk operation keys
	op_rgb      = 0b11111110 // RGB pixel data
	op_rgba     = 0b11111111 // RGBA pixel data
	op_index    = 0b00000000 // Color table pixel data
	op_diff     = 0b01000000 // Diff pixel data
	op_luma     = 0b10000000 // Another kind of diff pixel data
	op_run      = 0b11000000 // Begin a run of pixels
	op_key_mask = 0b11000000 // 2-bit mask for the operation key

	hashTableSize = 64
	headerSize    = 14
)

// TODO: init & register the image format or reader or something?

func Decode(r io.Reader) (image.Image, error) {
	// Read entire reader into byte slice
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	h, err := readHeader([14]byte(data[:headerSize]))
	if err != nil {
		return nil, err
	}

	img := image.NewNRGBA(image.Rect(0, 0, int(h.width), int(h.height)))
	colorTable := [hashTableSize]color.NRGBA{}

	previousPixel := color.NRGBA{0, 0, 0, 255}
	run := 0
	i := headerSize

	nextByte := func() byte {
		b := data[i]
		i++
		return b
	}

	for y := 0; y < int(h.height); y++ {
		for x := 0; x < int(h.width); x++ {
			pixel := previousPixel // Default to previous pixel

			if run > 0 { // Continue the run
				run--
			} else { // Begin a new operation
				opKey := nextByte()

				if opKey == op_rgb || opKey == op_rgba { // Normal pixel data
					pixel.R = nextByte()
					pixel.G = nextByte()
					pixel.B = nextByte()

					if opKey == op_rgba {
						pixel.A = nextByte()
					}
				} else if (opKey & op_key_mask) == op_index { // Color table lookup
					pixel = colorTable[(int(opKey) & ^op_key_mask)]
				} else if (opKey & op_key_mask) == op_diff { // Simple diff from previous pixel (alpha unchanged)
					pixel.R += ((opKey >> 4) & 0b00000011) - 2
					pixel.G += ((opKey >> 2) & 0b00000011) - 2
					pixel.B += ((opKey >> 0) & 0b00000011) - 2
				} else if (opKey & op_key_mask) == op_luma { // More complex diff from previous pixel (alpha unchanged)
					diffGreen := (opKey & ^byte(op_key_mask)) - 32
					diff := nextByte()

					pixel.R += diffGreen - 8 + ((diff >> 4) & 0b00001111)
					pixel.G += diffGreen
					pixel.B += diffGreen - 8 + ((diff >> 0) & 0b00001111)
				} else if (opKey & op_key_mask) == op_run { // Begin a run of pixels
					run = int(opKey) & ^op_key_mask
				} else { // Invalid operation key. I don't think we should ever hit this, but we'll return the image in progress, if so...
					return img, errors.New("qoi: invalid operation key")
				}

				// Add the pixel to the color table
				hashTableIndex := ((int(pixel.R) * 3) + (int(pixel.G) * 5) + (int(pixel.B) * 7) + (int(pixel.A) * 11)) % hashTableSize
				colorTable[hashTableIndex] = pixel
			}

			img.Set(x, y, pixel)  // Set the pixel in the image
			previousPixel = pixel // Update the previous pixel
		}
	}

	return img, nil
}
