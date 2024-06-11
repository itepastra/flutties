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
	SizeX    uint16
	SizeY    uint16
	Cells    []color.Color
	Modified time.Time
}

func NewGrid(sizeX uint16, sizeY uint16, defaultValue color.Color) Grid {
	grid := Grid{
		SizeX:    sizeX,
		SizeY:    sizeY,
		Cells:    make([]color.Color, (uint32(sizeX) * uint32(sizeY))),
		Modified: time.Now(),
	}
	for i := range grid.Cells {
		grid.Cells[i] = defaultValue
	}
	return grid
}

func randomColor() color.Color {
	cint := rand.Uint32()
	return color.RGBA{
		R: uint8(cint >> 24),
		G: uint8(cint >> 16),
		B: uint8(cint >> 8),
		A: 255,
	}
}
func NewGridRandom(sizeX uint16, sizeY uint16) Grid {
	grid := Grid{
		SizeX:    sizeX,
		SizeY:    sizeY,
		Cells:    make([]color.Color, (uint32(sizeX) * uint32(sizeY))),
		Modified: time.Now(),
	}
	for i := range grid.Cells {
		grid.Cells[i] = randomColor()
	}
	return grid
}

func (g *Grid) Get(x uint16, y uint16) (color.Color, error) {
	idx := int(y)*int(g.SizeX) + int(x)
	if idx >= len(g.Cells) {
		return nil, errors.New("Out of bounds")
	}
	return g.Cells[idx], nil
}

func blend(originalColor color.Color, newColor color.Color) color.Color {
	or, og, ob, _ := originalColor.RGBA()
	nr, ng, nb, a := newColor.RGBA()
	s := 255 - a
	return color.RGBA{
		R: uint8((or*s + nr*a) / 0xff),
		G: uint8((og*s + ng*a) / 0xff),
		B: uint8((ob*s + nb*a) / 0xff),
		A: 255,
	}
}

func (g *Grid) Set(x uint16, y uint16, c color.Color) error {
	idx := int(y)*int(g.SizeX) + int(x)
	if idx >= len(g.Cells) {
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
	color, _ := g.Get(uint16(x), uint16(y))
	return color
}
