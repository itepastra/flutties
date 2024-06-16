package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"image/jpeg"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/textproto"
	"time"

	"github.com/a-h/templ"
	"github.com/gorilla/websocket"
	"github.com/itepastra/flutties/helpers"
	"github.com/itepastra/flutties/helpers/multi"
	"github.com/itepastra/flutties/pages"
	"github.com/itepastra/flutties/types"
)

const (
	TIMEOUT_DELAY       = 30 * time.Second
	BOUNDARY_STRING     = "thisisaboundary"
	BOUNDARY_STRING_ICO = "thisisicoboundary"
	JPEG_UPDATE_TIMER   = 25 * time.Millisecond
	JPEG_PING_TIMER     = 25 * time.Second
	ICON_UPDATE_TIMER   = 25 * time.Millisecond
	ICON_WIDTH          = 32
	ICON_HEIGHT         = 32
	STATS_UPDATE_TIMER  = 200 * time.Millisecond
)

var upgrader = websocket.Upgrader{}
var (
	pixelflut_port          = flag.String("pixelflut", ":7791", "the port where the pixelflut is accessible internally")
	pixelflut_port_external = flag.String("pixelflut_ext", "55282", "the port where the pixelflut is accessible externally, used for the webpage")
	web_port                = flag.String("web", ":7792", "the address the website should listen on")
	width                   = flag.Uint("width", 800, "the canvas width")
	height                  = flag.Uint("height", 600, "the canvas height")
)

var (
	changedPixels  [types.GRID_AMOUNT]uint64 = [types.GRID_AMOUNT]uint64{}
	currentClients uint
)

const (
	INFO            byte = helpers.INFO
	SIZE                 = helpers.SIZE
	GET_PIXEL_VALUE      = helpers.GET_PIXEL_VALUE
	SET_GRAYSCALE        = helpers.SET_GRAYSCALE
	SET_HALF_RGBA        = helpers.SET_HALF_RGBA
	SET_RGB              = helpers.SET_RGB
	SET_RGBA             = helpers.SET_RGBA
	SOUND_LOOP           = helpers.SOUND_LOOP
	SOUND_ONCE           = helpers.SOUND_ONCE
	H                    = helpers.H
	I                    = helpers.I
	P                    = helpers.P
	S                    = helpers.S
)

func helpMsg() []byte {
	return []byte(`This is pixelgo server
it implements the pixelflut protocol :3
idk what to write here yet`)
}

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func CheckMinLength(data []byte, targetLen int) (advance int, token []byte, err error) {
	if len(data) < targetLen {
		return 0, nil, nil
	} else {
		return targetLen, data[:targetLen], nil
	}
}

func ScanCommands(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if data[0]&0xf0 == SET_RGB {
		return CheckMinLength(data, 8)
	}

	switch data[0] & 0xf0 {
	case INFO:
		return CheckMinLength(data, 1)
	case SIZE:
		return CheckMinLength(data, 1)
	case GET_PIXEL_VALUE:
		return CheckMinLength(data, 5)
	case SET_GRAYSCALE:
		return CheckMinLength(data, 6)
	case SET_HALF_RGBA:
		return CheckMinLength(data, 7)
	case SET_RGBA:
		return CheckMinLength(data, 9)
	case SOUND_LOOP:
		return CheckMinLength(data, 5)
	case SOUND_ONCE:
		return CheckMinLength(data, 5)
	case H & 0xf0: // same as I
	case P & 0xf0: // same as S
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			// We have a full newline-terminated line.
			return i + 1, dropCR(data[0:i]), nil
		}
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return 0, nil, nil
	}
	// Request more data.
	return 0, nil, nil
}

func handleConnection(conn net.Conn, grids [types.GRID_AMOUNT]*types.Grid) {
	currentClients = currentClients + 1
	defer func() {
		currentClients = currentClients - 1
		conn.Close()
	}()
	c := bufio.NewScanner(conn)
	c.Split(ScanCommands)
	for {
		if !c.Scan() {
			if c.Err() == nil {
				return
			}
			log.Printf("Got error from scanner %e", c.Err())
		}
		var err error
		cmd := c.Bytes()
		instruction := cmd[0]
		if instruction == H || instruction == I || instruction == P || instruction == S {
			err = helpers.TextCmd(cmd, grids, conn, &changedPixels)
		} else {
			err = helpers.BinCmd(cmd, grids, conn, &changedPixels)
		}
		if err != nil {
			log.Printf("connection %v had an error %e while sending, disconnecting", conn, err)
			return
		}
	}

}

func frameGenerator(grid *types.Grid, multiWriter multi.MapWriter, ch <-chan struct{}) {
	multipartWriter := multipart.NewWriter(multiWriter)
	multipartWriter.SetBoundary(BOUNDARY_STRING)
	header := make(textproto.MIMEHeader)
	header.Add("Content-Type", "image/jpeg")
	for {
		select {
		case <-ch:
			writer, _ := multipartWriter.CreatePart(header)
			jpeg.Encode(writer, grid, &jpeg.Options{Quality: 75})
		}
	}
}

func frameTimer(grid *types.Grid, ch chan<- struct{}) {
	for {
		ch <- struct{}{}
		time.Sleep(JPEG_UPDATE_TIMER)
		if time.Since(grid.Modified) > JPEG_UPDATE_TIMER*2 {
			mt := grid.Modified
			for mt == grid.Modified && time.Since(grid.Modified) < JPEG_PING_TIMER {
				time.Sleep(JPEG_UPDATE_TIMER)
			}
			grid.Modified = time.Now()
		}
	}
}

func main() {
	flag.Parse()

	multiWriter := multi.NewMapWriter()

	grid := types.NewGridRandom(uint16(*width), uint16(*height))
	icoGrid := types.NewGridRandom(ICON_WIDTH, ICON_HEIGHT)

	ln, err := net.Listen("tcp", *pixelflut_port)
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Printf("pixelflut started listening at %s, the grid has size (%d, %d)", *pixelflut_port, *width, *height)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("rip connection: %e", err)
			}
			go handleConnection(conn, [types.GRID_AMOUNT]*types.Grid{&grid, &icoGrid})
		}
	}()

	ch := make(chan struct{})

	go frameGenerator(&grid, multiWriter, ch)
	go frameTimer(&grid, ch)

	http.Handle("/", templ.Handler(pages.Index(*pixelflut_port_external)))
	http.HandleFunc("/icoflut.js", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "./static/icoflut.js") })
	http.HandleFunc("/icoflut", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %e", err)
			return
		}
		defer c.Close()
		for {
			writer, err := c.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return
			}
			jpeg.Encode(writer, &icoGrid, &jpeg.Options{Quality: 90})
			time.Sleep(ICON_UPDATE_TIMER)
			if time.Since(icoGrid.Modified) > ICON_UPDATE_TIMER {
				mt := icoGrid.Modified
				for mt == icoGrid.Modified {
					time.Sleep(ICON_UPDATE_TIMER)
				}
			}
		}
	})
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %e", err)
			return
		}
		defer c.Close()
		for {
			writer, err := c.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			writer.Write([]byte(fmt.Sprintf(`{"c":%d,"p":%d,"i":%d}`, currentClients, changedPixels[0], changedPixels[1])))
			time.Sleep(STATS_UPDATE_TIMER)
		}
	})
	http.HandleFunc("/icon", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpg")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Connection", "close")
		jpeg.Encode(w, &icoGrid, &jpeg.Options{Quality: 90})
	})
	http.HandleFunc("/grid", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(
			"Content-Type",
			fmt.Sprintf("multipart/x-mixed-replace;boundary=%s", BOUNDARY_STRING),
		)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Connection", "close")

		multiWriter.Add(w)
		ch <- struct{}{}
		ch <- struct{}{} // NOTE: firefox does not load the bottom rows correctly without this
		<-r.Context().Done()
		multiWriter.Remove(w)
	})

	log.Fatal(http.ListenAndServe(*web_port, nil))
}
