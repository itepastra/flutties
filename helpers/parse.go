package helpers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"io"
	"strconv"

	"github.com/itepastra/flutties/types"
)

var helpMessage = []byte(`
This is flutties, a pixelflut server written in go.
It supports the pixelflut protocol,
and also the binary protocol from the README
`)

const (
	INFO            byte = 0x1
	SIZE                 = 0x2
	GET_PIXEL_VALUE      = 0x8
	SET_GRAYSCALE        = 0x9
	SET_HALF_RGBA        = 0xA
	SET_RGB              = 0xB
	SET_RGBA             = 0xC
	SOUND_LOOP           = 0xE
	SOUND_ONCE           = 0xF
	TEXT_1               = 0x4
	TEXT_2               = 0x5
)

var (
	HELP_COMMAND          = []byte("HELP")
	SIZE_COMMAND          = []byte("SIZE")
	SIZE_ICON_COMMAND     = []byte("ISIZE")
	PX_COMMAND_START      = []byte("PX ")
	PX_ICON_COMMAND_START = []byte("IPX ")
	MAIN_GRID_INDEX       = 0
	ICON_GRID_INDEX       = 1
)

func parseHex(part string) (color.Color, error) {
	data, err := hex.DecodeString(part)
	if err != nil {
		return nil, err
	}
	if len(data) == 1 {
		return color.RGBA{
			R: data[0],
			G: data[0],
			B: data[0],
			A: 255,
		}, nil
	}
	if len(data) == 3 {
		return color.RGBA{
			R: data[0],
			G: data[1],
			B: data[2],
			A: 255,
		}, nil
	}
	if len(data) == 4 {
		return color.RGBA{
			R: data[0],
			G: data[1],
			B: data[2],
			A: data[3],
		}, nil
	}
	return nil, errors.New("incorrect number of bytes")
}

func parsePx(command []byte) (x uint16, y uint16, color color.Color, err error) {
	str := command
	trimmed := bytes.Trim(str, "\n \x00")
	parts := bytes.Split(trimmed, []byte{' '})
	var xu uint64
	var yu uint64
	for i, p := range parts {
		switch i {
		case 0:
			xu, err = strconv.ParseUint(string(p), 10, 16)
			x = uint16(xu)
			if err != nil {
				return
			}
		case 1:
			yu, err = strconv.ParseUint(string(p), 10, 16)
			y = uint16(yu)
			if err != nil {
				return
			}
		case 2:
			color, err = parseHex(string(p))
			if err != nil {
				return
			}
		}
	}
	return
}

func BinCmd(cmd []byte, grids [types.GRID_AMOUNT]*types.Grid, writer io.Writer, changedPixels *[types.GRID_AMOUNT]uint) (err error) {
	canvasId := cmd[0] & 0x0f
	switch cmd[0] >> 4 {
	case INFO:
		_, err = writer.Write([]byte(fmt.Sprintf("There are %d grids", len(grids))))
	case SIZE:
		_, err = writer.Write([]byte{
			cmd[0],
			byte(grids[canvasId].SizeX >> 8),
			byte(grids[canvasId].SizeX),
			byte(grids[canvasId].SizeY >> 8),
			byte(grids[canvasId].SizeY),
		})
	case GET_PIXEL_VALUE:
		x := uint16(cmd[2]) | uint16(cmd[1])<<8
		y := uint16(cmd[4]) | uint16(cmd[3])<<8
		color, err := grids[canvasId].Get(x, y)
		r, g, b, _ := color.RGBA()
		if err != nil {
			return err
		}
		_, err = writer.Write([]byte{
			cmd[0],
			cmd[1],
			cmd[2],
			cmd[3],
			cmd[4],
			byte(r),
			byte(g),
			byte(b),
		})
	case SET_GRAYSCALE:
		x := uint16(cmd[2]) | uint16(cmd[1])<<8
		y := uint16(cmd[4]) | uint16(cmd[3])<<8
		g := cmd[5]
		err = grids[canvasId].Set(x, y, color.RGBA{R: g, G: g, B: g, A: 255})
	case SET_HALF_RGBA:
		x := uint16(cmd[2]) | uint16(cmd[1])<<8
		y := uint16(cmd[4]) | uint16(cmd[3])<<8
		r := (cmd[5] & 0xf0) | (cmd[5]&0xf0)>>4
		g := (cmd[5]&0x0f)<<4 | (cmd[5] & 0x0f)
		b := (cmd[6] & 0xf0) | (cmd[6]&0xf0)>>4
		a := (cmd[6]&0x0f)<<4 | (cmd[6] & 0x0f)
		err = grids[canvasId].Set(x, y, color.RGBA{R: r, G: g, B: b, A: a})
	case SET_RGB:
		x := uint16(cmd[2]) | uint16(cmd[1])<<8
		y := uint16(cmd[4]) | uint16(cmd[3])<<8
		r := cmd[5]
		g := cmd[6]
		b := cmd[7]
		err = grids[canvasId].Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
	case SET_RGBA:
		x := uint16(cmd[2]) | uint16(cmd[1])<<8
		y := uint16(cmd[4]) | uint16(cmd[3])<<8
		r := cmd[5]
		g := cmd[6]
		b := cmd[7]
		a := cmd[8]
		err = grids[canvasId].Set(x, y, color.RGBA{R: r, G: g, B: b, A: a})
	}
	return
}

func TextCmd(cmd []byte, grids [types.GRID_AMOUNT]*types.Grid, writer io.Writer, changedPixels *[types.GRID_AMOUNT]uint) (err error) {
	if bytes.Compare(cmd, HELP_COMMAND) == 0 {
		_, err = writer.Write(helpMessage)
	} else if bytes.Compare(cmd, SIZE_COMMAND) == 0 {
		_, err = writer.Write([]byte(fmt.Sprintf("SIZE %d %d\n", grids[0].SizeX, grids[0].SizeY)))
	} else if bytes.Compare(cmd, SIZE_ICON_COMMAND) == 0 {
		_, err = writer.Write([]byte(fmt.Sprintf("SIZE %d %d\n", grids[1].SizeX, grids[1].SizeY)))
	} else if rest, found := bytes.CutPrefix(cmd, PX_COMMAND_START); found {
		x, y, color, err := parsePx(rest)
		if err != nil {
			return err
		}
		if color == nil { // a request for the current color
			c, err := grids[MAIN_GRID_INDEX].Get(x, y)
			if err != nil {
				return err
			}
			_, err = writer.Write([]byte(fmt.Sprintf("PX %d %d %s\n", x, y, PxToHex(c))))
		} else {
			err = grids[MAIN_GRID_INDEX].Set(x, y, color)
			changedPixels[0]++
		}
	} else if rest, found := bytes.CutPrefix(cmd, PX_ICON_COMMAND_START); found {
		x, y, color, err := parsePx(rest)
		if err != nil {
			return err
		}
		if color == nil { // a request for the current color
			c, err := grids[ICON_GRID_INDEX].Get(x, y)
			if err != nil {
				return err
			}
			_, err = writer.Write([]byte(fmt.Sprintf("PX %d %d %s\n", x, y, PxToHex(c))))
		} else {
			err = grids[ICON_GRID_INDEX].Set(x, y, color)
			changedPixels[1]++
		}
	}
	return
}
