package qoi

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type header struct {
	magic      [4]byte
	width      uint32
	height     uint32
	channels   uint8
	colorspace uint8
}

func readHeader(data [14]byte) (header, error) {
	var h header

	if err := binary.Read(bytes.NewReader(data[:]), binary.BigEndian, &h); err != nil {
		return h, err
	}

	if string(h.magic[:]) != "qoif" { // Check for the magic string
		return h, errors.New("qoi: invalid magic string")
	}

	if h.width == 0 || h.height == 0 { // Check for valid dimensions
		return h, errors.New("qoi: invalid image dimensions")
	}

	if h.channels != 3 && h.channels != 4 { // Check for valid number of channels
		return h, errors.New("qoi: invalid number of channels")
	}

	return h, nil
}
