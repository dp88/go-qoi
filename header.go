package qoi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
)

type header struct {
	Width      uint32
	Height     uint32
	Channels   uint8
	Colorspace uint8
}

func readHeader(data [14]byte) (h header, err error) {
	if err = binary.Read(bytes.NewReader(data[4:]), binary.BigEndian, &h); err != nil {
		return h, err
	}

	// Check for the magic string
	if string(data[:4]) != "qoif" {
		return h, errors.New("qoi: invalid magic string")
	}

	// Check for valid dimensions
	if h.Width == 0 || h.Height == 0 {
		return h, errors.New("qoi: invalid image dimensions")
	}

	// Check for valid number of channels
	if h.Channels != 3 && h.Channels != 4 {
		return h, errors.New("qoi: invalid number of channels")
	}

	// Check for valid colorspace
	if h.Colorspace > 1 {
		return h, errors.New("qoi: invalid colorspace")
	}

	return h, nil
}

func (h header) AsConfig() image.Config {
	return image.Config{
		ColorModel: color.NRGBAModel,
		Width:      int(h.Width),
		Height:     int(h.Height),
	}
}
