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

func TextCmd(cmd []byte, grids [types.GRID_AMOUNT]*types.Grid, writer io.Writer) (err error) {
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
		}
	}
	return
}
