package qoi

import (
	"errors"
)

type header struct {
	width      uint32
	height     uint32
	channels   uint8
	colorspace uint8
}

func readHeader(data [14]byte) (header, error) {
	var h header

	// Check for the magic string
	if string(data[:4]) != "qoif" {
		return h, errors.New("qoi: invalid magic string")
	}

	// Read the width and height
	h.width = uint32(data[4])<<24 | uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7])
	h.height = uint32(data[8])<<24 | uint32(data[9])<<16 | uint32(data[10])<<8 | uint32(data[11])

	if h.width == 0 || h.height == 0 {
		return h, errors.New("qoi: invalid image dimensions")
	}

	h.channels = data[12]
	h.colorspace = data[13]

	if h.channels != 3 && h.channels != 4 {
		return h, errors.New("qoi: invalid number of channels")
	}

	return h, nil
}
