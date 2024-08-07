package qoi

import (
	"bufio"
	"encoding/binary"
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

func init() {
	image.RegisterFormat("qoi", "qoif", Decode, DecodeConfig)
}

func DecodeConfig(r io.Reader) (image.Config, error) {
	var headerBytes [headerSize]byte
	// new buffer to read bytes from r
	br := bufio.NewReader(r)

	for i := 0; i < headerSize; i++ {
		b, err := br.ReadByte()
		if err != nil {
			return image.Config{}, err
		}
		headerBytes[i] = b
	}

	h, err := readHeader(headerBytes)
	if err != nil {
		return image.Config{}, err
	}

	return h.AsConfig(), nil
}

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
				colorTable[hashTableIndex(pixel)] = pixel
			}

			img.Set(x, y, pixel)  // Set the pixel in the image
			previousPixel = pixel // Update the previous pixel
		}
	}

	return img, nil
}

func Encode(w io.Writer, img image.Image) error {
	bw := bufio.NewWriter(w)

	bw.WriteString("qoif")

	h := header{
		width:      uint32(img.Bounds().Dx()),
		height:     uint32(img.Bounds().Dy()),
		channels:   4,
		colorspace: 0,
	}

	if err := binary.Write(bw, binary.BigEndian, h); err != nil {
		return err
	}

	colorTable := [hashTableSize]color.NRGBA{}
	previousPixel := color.NRGBA{0, 0, 0, 255}
	run := 0

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	flushRun := func() {
		bw.WriteByte(op_run | byte(run-1)) // -1 because a run value of 0 is a single pixel
		run = 0
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)

			// Check if we're in a run of pixels
			if pixel == previousPixel {
				run++

				if run == 62 || (x == width-1 && y == height-1) { // End of a run
					flushRun()
				}
			} else { // Just regular pixel data
				if run > 0 {
					flushRun()
				}

				// Check if the pixel is in the color table
				tableIndex := hashTableIndex(pixel)
				if colorTable[tableIndex] == pixel {
					bw.WriteByte(op_index | byte(tableIndex))
				} else {
					// Write the pixel to the color table
					colorTable[tableIndex] = pixel

					bw.WriteByte(op_rgba)
					bw.WriteByte(pixel.R)
					bw.WriteByte(pixel.G)
					bw.WriteByte(pixel.B)
					bw.WriteByte(pixel.A)
				}

				previousPixel = pixel
			}
		}
	}

	// End of QOI image
	for i := 0; i < 7; i++ {
		bw.WriteByte(0)
	}
	bw.WriteByte(0x01)

	return bw.Flush()
}

func hashTableIndex(c color.NRGBA) int {
	return ((int(c.R) * 3) + (int(c.G) * 5) + (int(c.B) * 7) + (int(c.A) * 11)) % hashTableSize
}
