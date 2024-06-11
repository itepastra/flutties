package types

import (
	"errors"
	"image"
	"image/color"
	"math/rand/v2"
	"time"
)

type Grid struct {
	SizeX    int
	SizeY    int
	Cells    []color.Color
	Modified time.Time
}

func NewGrid(sizeX int, sizeY int, defaultValue color.Color) Grid {
	grid := Grid{
		SizeX:    sizeX,
		SizeY:    sizeY,
		Cells:    make([]color.Color, (sizeX * sizeY)),
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
func NewGridRandom(sizeX int, sizeY int) Grid {
	grid := Grid{
		SizeX:    sizeX,
		SizeY:    sizeY,
		Cells:    make([]color.Color, (sizeX * sizeY)),
		Modified: time.Now(),
	}
	for i := range grid.Cells {
		grid.Cells[i] = randomColor()
	}
	return grid
}

func (g *Grid) Get(x int, y int) (color.Color, error) {
	idx := y*g.SizeX + x
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

func (g *Grid) Set(x int, y int, c color.Color) error {
	idx := y*g.SizeX + x
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
	return image.Rect(0, 0, g.SizeX, g.SizeY)
}

func (g *Grid) At(x, y int) color.Color {
	color, _ := g.Get(x, y)
	return color
}
