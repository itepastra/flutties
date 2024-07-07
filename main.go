package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/textproto"
	"strconv"
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

func createScanCommands(grids [types.GRID_AMOUNT]*types.Grid, conn io.Writer) func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		switch data[0] & 0xf0 {
		case INFO:
			return helpers.HelpBin(conn, data)
		case SIZE:
			return helpers.InfoBin(conn, data, grids)
		case GET_PIXEL_VALUE:
			return helpers.GetPixelBin(conn, data, grids)
		case SET_GRAYSCALE:
			return helpers.SetGrayscaleBin(data, grids)
		case SET_HALF_RGBA:
			return helpers.SetHalfRGBABin(data, grids)
		case SET_RGB:
			return helpers.SetRGBBin(data, grids)
		case SET_RGBA:
			return helpers.SetRGBABin(data, grids)
		case H & 0xf0:
		case P & 0xf0:
			if i := bytes.IndexByte(data, '\n'); i >= 0 {
				dropped := dropCR(data[:i])
				helpers.TextCmd(dropped, grids, conn)
				return i + i, dropped, nil
			}
		}

		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return 0, nil, nil
		}
		// Request more data.
		return 0, nil, nil
	}
}

func handleConnection(conn net.Conn, grids [types.GRID_AMOUNT]*types.Grid) {
	currentClients = currentClients + 1
	defer func() {
		currentClients = currentClients - 1
		conn.Close()
	}()
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in handleConnection: ", r)
		}
	}()
	c := bufio.NewScanner(conn)
	c.Split(createScanCommands(grids, conn))
	for {
		if !c.Scan() {
			if c.Err() == nil {
				return
			}
			log.Printf("Got error from scanner %e", c.Err())
		}
		var err error
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
	prev := grid.ChangedPixels
	for {
		ch <- struct{}{}
		time.Sleep(JPEG_UPDATE_TIMER)
		if prev == grid.ChangedPixels {
			// nothing happened since last update. since no new pixels
			for prev == grid.ChangedPixels && time.Since(grid.Modified) < JPEG_PING_TIMER {
				time.Sleep(JPEG_UPDATE_TIMER)
			}
			grid.Modified = time.Now()
		} else {
			grid.Modified = time.Now()
		}
	}
}

func drawShape(grid *types.Grid, dc drawcall) error {
	color_a, err := strconv.ParseInt(dc.Color, 16, 32)
	if err != nil {
		return err
	}
	color := uint32((color_a&0xff)<<16 | color_a&0xff00 | (color_a&0xff0000)>>16 | (0xff << 24))
	for i := -dc.Size; i <= dc.Size; i += 1 {
		if dc.Y+i < 0 {
			continue
		}
		if dc.Y+i >= grid.SizeY {
			break
		}
		for j := -dc.Size; j <= dc.Size; j += 1 {
			if dc.X+j < 0 {
				continue
			}
			if dc.X+j >= grid.SizeX {
				break
			}
			if i*i+j*j >= dc.Size*dc.Size {
				continue
			}
			err := grid.SetExact(uint32(i+dc.Y)<<16|uint32(j+dc.X), color)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type drawcall struct {
	X     int
	Y     int
	Color string
	Size  int
}

func main() {
	flag.Parse()

	multiWriter := multi.NewMapWriter()

	grid := types.NewGridRandom(uint16(*width), uint16(*height), 0)
	icoGrid := types.NewGridRandom(ICON_WIDTH, ICON_HEIGHT, 1)

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
	http.HandleFunc("/icoflut.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/javascript")
		http.ServeFile(w, r, "./static/icoflut.js")
	})
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
		// keep writing the stats to the websocket
		go func() {
			for {
				writer, err := c.NextWriter(websocket.TextMessage)
				if err != nil {
					return
				}
				writer.Write([]byte(fmt.Sprintf(`{"c":%d,"p":%d,"i":%d}`, currentClients, grid.ChangedPixels, icoGrid.ChangedPixels)))
				time.Sleep(STATS_UPDATE_TIMER)
			}
		}()
		// receive draw calls from the websocket
		for {
			_, data, err := c.ReadMessage()
			if err != nil {
				return
			}
			log.Println(string(data))
			drawCall := drawcall{}
			err = json.Unmarshal(data, &drawCall)
			if err != nil {
				log.Println(err)
				return
			}
			drawShape(&grid, drawCall)
		}
	})
	http.HandleFunc("/icon", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
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

	http.HandleFunc("/color/{x}/{y}/{color}/{size}", func(w http.ResponseWriter, r *http.Request) {
		xStr := r.PathValue("x")
		x, err := strconv.Atoi(xStr)
		if err != nil {
			log.Println("x parse err")
			return
		}
		yStr := r.PathValue("y")
		y, err := strconv.Atoi(yStr)
		if err != nil {
			log.Println("y parse err")
			return
		}
		colorStr := r.PathValue("color")
		color, err := strconv.ParseInt(colorStr, 16, 32)
		if err != nil {
			log.Println("color parse err")
			return
		}
		color = (color&0xff)<<16 | color&0xff00 | (color&0xff0000)>>16 | (0xff << 24)
		log.Println(color)

		sizeStr := r.PathValue("size")
		size, err := strconv.Atoi(sizeStr)
		if err != nil {
			log.Println("size parse err")
			return
		}

		for i := -size; i <= size; i += 1 {
			if y+i < 0 {
				continue
			}
			if y+i >= grid.SizeY {
				break
			}
			for j := -size; j <= size; j += 1 {
				if x+j < 0 {
					continue
				}
				if x+j >= grid.SizeX {
					break
				}
				if i*i+j*j >= size*size {
					continue
				}
				err = grid.SetExact(uint32(i+y)<<16|uint32(j+x), uint32(color))
				if err != nil {
					log.Println("oop")
					return
				}
			}
		}

	})

	log.Fatal(http.ListenAndServe(*web_port, nil))
}
