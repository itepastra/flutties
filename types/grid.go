package types

import (
	"errors"
	"image"
	"image/color"
	"math/rand/v2"
	"time"
)

const GRID_AMOUNT = 2

type Grid struct {
	SizeX    int
	SizeY    int
	length   int
	Cells    []uint32
	Modified time.Time
}

func NewGrid(sizeX uint16, sizeY uint16, defaultValue uint32) Grid {
	grid := Grid{
		SizeX:    int(sizeX),
		SizeY:    int(sizeY),
		length:   int(sizeX) * int(sizeY),
		Cells:    make([]uint32, (uint32(sizeX) * uint32(sizeY))),
		Modified: time.Now(),
	}
	for i := range grid.Cells {
		grid.Cells[i] = defaultValue
	}
	return grid
}

func randomColor() uint32 {
	return rand.Uint32() | 0xff
}

func NewGridRandom(sizeX uint16, sizeY uint16) Grid {
	grid := Grid{
		SizeX:    int(sizeX),
		SizeY:    int(sizeY),
		length:   int(sizeX) * int(sizeY),
		Cells:    make([]uint32, (uint32(sizeX) * uint32(sizeY))),
		Modified: time.Now(),
	}
	for i := range grid.Cells {
		grid.Cells[i] = randomColor()
	}
	return grid
}

func (g *Grid) Get(x uint16, y uint16) (uint32, error) {
	idx := int(y)*int(g.SizeX) + int(x)
	if idx >= len(g.Cells) {
		return 0, errors.New("Out of bounds")
	}
	return g.Cells[idx], nil
}

func blend(src uint32, dst uint32) uint32 {
	t := dst & 0xff
	s := 255 - t
	return (((((src>>24)&0xff)*s +
		((dst>>24)&0xff)*t) << 16) & 0xff000000) |
		(((((src>>16)&0xff)*s +
			((dst>>16)&0xff)*t) << 8) & 0xffff0000) |
		((((src>>8)&0xff)*s +
			((dst>>8)&0xff)*t) & 0xffffff00) |
		0xff
}

func (g *Grid) Set(xy uint32, c uint32) error {
	idx := int(xy&0xffff)*int(g.SizeX) + int(xy>>16)
	if idx >= g.length {
		return errors.New("Out of bounds")
	}
	g.Cells[idx] = blend(g.Cells[idx], c)
	g.Modified = time.Now()
	return nil
}

func (g *Grid) ColorModel() color.Model {
	return color.RGBAModel
}

func (g *Grid) Bounds() image.Rectangle {
	return image.Rect(0, 0, int(g.SizeX), int(g.SizeY))
}

func (g *Grid) At(x, y int) color.Color {
	c, _ := g.Get(uint16(x), uint16(y))
	return color.RGBA{R: byte(c >> 24), G: byte(c >> 16), B: byte(c >> 8), A: byte(c)}
}
