package main

import (
	"bufio"
	"flag"
	"fmt"
	"image/jpeg"
	"log"
	"mime/multipart"
	"net"
	"net/http"
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
)

var upgrader = websocket.Upgrader{}
var (
	cpuprofile              = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile              = flag.String("memprofile", "", "write memory profile to `file`")
	pixelflut_port          = flag.String("pixelflut", ":7791", "the port where the pixelflut is accessible internally")
	pixelflut_port_external = flag.String("pixelflut_ext", "55282", "the port where the pixelflut is accessible externally, used for the webpage")
	web_port                = flag.String("web", ":7792", "the address the website should listen on")
	width                   = flag.Uint("width", 800, "the canvas width")
	height                  = flag.Uint("height", 600, "the canvas height")
)

func byteComp(a []byte, b []byte) bool {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func timeTillDestroy() time.Time {
	return time.Now().Add(TIMEOUT_DELAY)
}

func helpMsg() []byte {
	return []byte(`This is pixelgo server
it implements the pixelflut protocol :3
idk what to write here yet`)
}

func handleConnection(conn net.Conn, grids [types.GRID_AMOUNT]*types.Grid) {
	log.Printf("started connection %v", conn)
	defer func() {
		log.Printf("stopped connection %v", conn)
		conn.Close()
	}()
	c := bufio.NewScanner(conn)
	for {
		if !c.Scan() {
			if c.Err() == nil {
				// Was EOF
				log.Printf("Got EOF from %v", conn)
				return
			}
			log.Printf("Got error from scanner %e", c.Err())
		}
		var err error
		cmd := c.Bytes()

		err = helpers.TextCmd(cmd, grids, conn)
		if err != nil {
			log.Printf("connection %v had an error %e while sending", conn, err)
			return
		}
	}

}

func frameGenerator(grid *types.Grid, multiWriter multi.MapWriter, ch <-chan byte) {
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

func frameTimer(grid *types.Grid, ch chan<- byte) {
	for {
		ch <- 0
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

	ch := make(chan byte)

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
		ch <- 1
		ch <- 1 // NOTE: firefox does not load the bottom rows correctly without this
		<-r.Context().Done()
		multiWriter.Remove(w)
	})

	log.Fatal(http.ListenAndServe(*web_port, nil))
}
