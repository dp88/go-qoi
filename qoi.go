package qoi

import (
	"bufio"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"

	"github.com/dp88/go-qoi/colortable"
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

	headerSize = 14
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
	return h.AsConfig(), err
}

func Decode(r io.Reader) (image.Image, error) {
	// Read entire reader into byte slice
	br := bufio.NewReader(r)
	cfg, err := DecodeConfig(br)
	if err != nil {
		return nil, err
	}

	img := image.NewNRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
	lookup := colortable.ColorTable{}

	previousPixel := color.NRGBA{0, 0, 0, 255}
	run := 0

	nextByte := func() byte {
		b, _ := br.ReadByte()
		return b
	}

	for y := 0; y < int(cfg.Height); y++ {
		for x := 0; x < int(cfg.Width); x++ {
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
					pixel = lookup.Get(int(opKey) & ^op_key_mask)
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
				lookup.Add(pixel)
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
		Width:      uint32(img.Bounds().Dx()),
		Height:     uint32(img.Bounds().Dy()),
		Channels:   4,
		Colorspace: 0,
	}

	if err := binary.Write(bw, binary.BigEndian, h); err != nil {
		return err
	}

	lookup := colortable.ColorTable{}
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
				if lookup.Contains(pixel) {
					bw.WriteByte(op_index | byte(colortable.IndexFor(pixel)))
				} else {
					// Write the pixel to the color table
					lookup.Add(pixel)

					if pixel.A == previousPixel.A {
						vr := int(pixel.R) - int(previousPixel.R)
						vg := int(pixel.G) - int(previousPixel.G)
						vb := int(pixel.B) - int(previousPixel.B)

						vgr := vr - vg
						vgb := vb - vg

						if vr > -3 && vr < 2 &&
							vg > -3 && vg < 2 &&
							vb > -3 && vb < 2 {
							bw.WriteByte(op_diff | byte((vr+2)<<4|(vg+2)<<2|(vb+2)<<0))
						} else if vgr > -9 && vgr < 8 &&
							vg > -33 && vg < 32 &&
							vgb > -9 && vgb < 8 {
							bw.WriteByte(op_luma | byte(vg+32))
							bw.WriteByte(byte(vgr+8)<<4 | byte(vgb+8)<<0)
						} else {
							bw.WriteByte(op_rgb)
							bw.WriteByte(pixel.R)
							bw.WriteByte(pixel.G)
							bw.WriteByte(pixel.B)
						}
					} else {
						bw.WriteByte(op_rgba)
						bw.WriteByte(pixel.R)
						bw.WriteByte(pixel.G)
						bw.WriteByte(pixel.B)
						bw.WriteByte(pixel.A)
					}
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
