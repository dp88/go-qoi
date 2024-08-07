package colortable

import "image/color"

const SIZE = 64

type ColorTable [SIZE]color.NRGBA

func IndexFor(c color.NRGBA) int {
	return ((int(c.R) * 3) + (int(c.G) * 5) + (int(c.B) * 7) + (int(c.A) * 11)) % SIZE
}

func (ct ColorTable) Contains(c color.NRGBA) bool {
	return ct[IndexFor(c)] == c
}

func (ct *ColorTable) Add(c color.NRGBA) {
	(*ct)[IndexFor(c)] = c
}

func (ct ColorTable) Get(index int) color.NRGBA {
	return ct[index]
}
