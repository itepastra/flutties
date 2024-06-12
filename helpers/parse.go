package helpers

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync/atomic"

	"github.com/itepastra/flutties/types"
)

var helpMessage = []byte(`
This is flutties, a pixelflut server written in go.
It supports the pixelflut protocol,
and also the binary protocol from the README
`)

const (
	INFO            byte = 0x10
	SIZE                 = 0x20
	GET_PIXEL_VALUE      = 0x80
	SET_GRAYSCALE        = 0x90
	SET_HALF_RGBA        = 0xA0
	SET_RGB              = 0xB0
	SET_RGBA             = 0xC0
	SOUND_LOOP           = 0xE0
	SOUND_ONCE           = 0xF0
	H                    = byte('H')
	I                    = byte('I')
	P                    = byte('P')
	S                    = byte('S')
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

func parseHex(part string) (uint32, error) {
	data, err := hex.DecodeString(part)
	if err != nil {
		return 0, err
	}
	if len(data) == 1 {
		return uint32(data[0])<<24 |
			uint32(data[0])<<16 |
			uint32(data[0])<<8 | 0xff, nil
	}
	if len(data) == 3 {
		return uint32(data[0])<<24 |
			uint32(data[1])<<16 |
			uint32(data[2])<<8 | 0xff, nil
	}
	if len(data) == 4 {
		return uint32(data[0])<<24 |
			uint32(data[1])<<16 |
			uint32(data[2])<<8 |
			uint32(data[3]), nil
	}
	return 0, errors.New("incorrect number of bytes")
}

func parsePx(command []byte) (x uint16, y uint16, found bool, color uint32, err error) {
	str := command
	trimmed := bytes.Trim(str, "\n \x00")
	parts := bytes.Split(trimmed, []byte{' '})
	var xu uint64
	var yu uint64
	found = false
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
			found = true
			if err != nil {
				return
			}
		}
	}
	return
}

func pack(a, b, c, d byte) uint32 {
	return uint32(a)<<24 | uint32(b)<<16 | uint32(c)<<8 | uint32(d)
}

func inc(changePixels *[types.GRID_AMOUNT]uint64, idx byte) {
	atomic.AddUint64(&changePixels[idx], 1)
}

func BinCmd(cmd []byte, grids [types.GRID_AMOUNT]*types.Grid, writer io.Writer, changedPixels *[types.GRID_AMOUNT]uint64) (err error) {
	canvasId := cmd[0] & 0x0f
	switch cmd[0] & 0xf0 {
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
		x := uint16(cmd[1])<<8 | uint16(cmd[2])
		y := uint16(cmd[3])<<8 | uint16(cmd[4])
		color, err := grids[canvasId].Get(x, y)
		if err != nil {
			return err
		}
		_, err = writer.Write([]byte{
			cmd[0],
			cmd[1],
			cmd[2],
			cmd[3],
			cmd[4],
			byte(color >> 24),
			byte(color >> 16),
			byte(color >> 8),
		})
	case SET_GRAYSCALE:
		err = grids[canvasId].Set(pack(cmd[1], cmd[2], cmd[3], cmd[4]), uint32(cmd[5])<<24|uint32(cmd[5])<<16|uint32(cmd[5])<<8|0xff)
		inc(changedPixels, canvasId)
	case SET_HALF_RGBA:
		r := (cmd[5] & 0xf0) | (cmd[5]&0xf0)>>4
		g := (cmd[5]&0x0f)<<4 | (cmd[5] & 0x0f)
		b := (cmd[6] & 0xf0) | (cmd[6]&0xf0)>>4
		a := (cmd[6]&0x0f)<<4 | (cmd[6] & 0x0f)
		err = grids[canvasId].Set(pack(cmd[1], cmd[2], cmd[3], cmd[4]), binary.BigEndian.Uint32([]byte{r, g, b, a}))
		inc(changedPixels, canvasId)
	case SET_RGB:
		err = grids[canvasId].Set(pack(cmd[1], cmd[2], cmd[3], cmd[4]), uint32(cmd[5])<<24|uint32(cmd[6])<<16|uint32(cmd[7])<<8|0xff)
		inc(changedPixels, canvasId)
	case SET_RGBA:
		err = grids[canvasId].Set(pack(cmd[1], cmd[2], cmd[3], cmd[4]), uint32(cmd[5])<<24|uint32(cmd[6])<<16|uint32(cmd[7])<<8|uint32(0xff))
		inc(changedPixels, canvasId)
	}
	return
}

func TextCmd(cmd []byte, grids [types.GRID_AMOUNT]*types.Grid, writer io.Writer, changedPixels *[types.GRID_AMOUNT]uint64) (err error) {
	if bytes.Compare(cmd, HELP_COMMAND) == 0 {
		_, err = writer.Write(helpMessage)
	} else if bytes.Compare(cmd, SIZE_COMMAND) == 0 {
		_, err = writer.Write([]byte(fmt.Sprintf("SIZE %d %d\n", grids[0].SizeX, grids[0].SizeY)))
	} else if bytes.Compare(cmd, SIZE_ICON_COMMAND) == 0 {
		_, err = writer.Write([]byte(fmt.Sprintf("SIZE %d %d\n", grids[1].SizeX, grids[1].SizeY)))
	} else if rest, found := bytes.CutPrefix(cmd, PX_COMMAND_START); found {
		x, y, found, color, err := parsePx(rest)
		if err != nil {
			return err
		}
		if !found { // a request for the current color
			c, err := grids[MAIN_GRID_INDEX].Get(x, y)
			if err != nil {
				return err
			}
			_, err = writer.Write([]byte(fmt.Sprintf("PX %d %d %s\n", x, y, PxToHex(c))))
		} else {
			err = grids[MAIN_GRID_INDEX].Set(uint32(x)<<16|uint32(y), color)
			inc(changedPixels, 0)
		}
	} else if rest, found := bytes.CutPrefix(cmd, PX_ICON_COMMAND_START); found {
		x, y, found, color, err := parsePx(rest)
		if err != nil {
			return err
		}
		if !found { // a request for the current color
			c, err := grids[ICON_GRID_INDEX].Get(x, y)
			if err != nil {
				return err
			}
			_, err = writer.Write([]byte(fmt.Sprintf("PX %d %d %s\n", x, y, PxToHex(c))))
		} else {
			err = grids[ICON_GRID_INDEX].Set(uint32(x)<<16|uint32(y), color)
			inc(changedPixels, 1)
		}
	}
	return
}
