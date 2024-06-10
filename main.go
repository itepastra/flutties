package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"image/jpeg"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"strconv"
	"time"

	"github.com/a-h/templ"
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
	CANVAS_WIDTH        = 800
	CANVAS_HEIGHT       = 600
	ICON_WIDTH          = 32
	ICON_HEIGHT         = 32
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

func parsePx(command []byte) (x uint64, y uint64, color color.Color, err error) {
	str := command
	trimmed := bytes.Trim(str, "\n \x00")
	parts := bytes.Split(trimmed, []byte{' '})
	for i, p := range parts {
		if i == 0 {
			continue
		}
		if i == 1 {
			x, err = strconv.ParseUint(string(p), 10, 0)
			if err != nil {
				return
			}
		}
		if i == 2 {
			y, err = strconv.ParseUint(string(p), 10, 0)
			if err != nil {
				return
			}
		}
		if i == 3 {
			color, err = parseHex(string(p))
			if err != nil {
				return
			}
		}
	}
	return
}

func handleConnection(conn net.Conn, grid *types.Grid, iconGrid *types.Grid) {
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

		if byteComp(cmd, []byte("HELP")) {
			_, err = conn.Write(helpMsg())
		} else if byteComp(cmd, []byte("SIZE")) {
			_, err = conn.Write([]byte(fmt.Sprintf("SIZE %d %d\n", grid.SizeX, grid.SizeY)))
		} else if byteComp(cmd, []byte("PX ")) {
			x, y, color, err := parsePx(cmd)
			if color == nil {
				c, err := grid.Get(int(x), int(y))
				if err != nil {
					log.Printf("Could not get color at (%d, %d)", x, y)
					return
				}
				_, err = conn.Write([]byte(fmt.Sprintf("PX %d %d %s\n", x, y, helpers.PxToHex(c))))
			} else {
				err := grid.Set(int(x), int(y), color)
				if err != nil {
					log.Printf("Could not set color at (%d, %d)", x, y)
					return
				}
			}
			if err != nil {
				log.Printf("PX format %s was not correct %e", cmd, err)
				return
			}
		} else if byteComp(cmd, []byte("ISIZE")) {
			_, err = conn.Write([]byte(fmt.Sprintf("ISIZE %d %d\n", iconGrid.SizeX, iconGrid.SizeY)))
		} else if byteComp(cmd, []byte("IPX ")) {
			x, y, color, err := parsePx(cmd)
			if color == nil {
				c, err := iconGrid.Get(int(x), int(y))
				if err != nil {
					log.Printf("Could not get color at (%d, %d)", x, y)
					return
				}
				_, err = conn.Write([]byte(fmt.Sprintf("IPX %d %d %s\n", x, y, helpers.PxToHex(c))))
			} else {
				err := iconGrid.Set(int(x), int(y), color)
				if err != nil {
					log.Printf("Could not set color at (%d, %d)", x, y)
					return
				}
			}
			if err != nil {
				log.Printf("IPX format %s was not correct %e", cmd, err)
				return
			}
		} else {
			return
		}
		if err != nil {
			log.Printf("connection %v had an error %e while sending", conn, err)
			return
		}
	}

}

func frameGenerator(grid *types.Grid, multiWriter multi.MapWriter) {
	multipartWriter := multipart.NewWriter(multiWriter)
	multipartWriter.SetBoundary(BOUNDARY_STRING)
	header := make(textproto.MIMEHeader)
	header.Add("Content-Type", "image/jpeg")
	for {
		writer, _ := multipartWriter.CreatePart(header)
		jpeg.Encode(writer, grid, &jpeg.Options{Quality: 75})
		time.Sleep(JPEG_UPDATE_TIMER)
	}
}

func main() {
	multiWriter := multi.NewMapWriter()
	icoWriter := multi.NewMapWriter()

	grid := types.NewGridRandom(CANVAS_WIDTH, CANVAS_HEIGHT)
	icoGrid := types.NewGridRandom(ICON_WIDTH, ICON_HEIGHT)

	ln, err := net.Listen("tcp", ":7791")
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Println("pixelflut started listening")
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("rip connection: %e", err)
			}
			go handleConnection(conn, &grid, &icoGrid)
		}
	}()

	http.Handle("/", templ.Handler(pages.Index()))
	//http.Handle("/grid", templ.Handler(pages.Grid(&grid)))

	go frameGenerator(&grid, multiWriter)
	go frameGenerator(&icoGrid, icoWriter)

	http.HandleFunc("/icoflut.js", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "./static/icoflut.js") })

	http.HandleFunc("/icon", func(w http.ResponseWriter, r *http.Request) {
		// w.Header().Set(
		// 	"Content-Type",
		// 	fmt.Sprintf("multipart/x-mixed-replace;boundary=%s", BOUNDARY_STRING_ICO),
		// )
		w.Header().Set("Content-Type", "image/jpg")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Connection", "close")
		jpeg.Encode(w, &icoGrid, &jpeg.Options{Quality: 100})

		// icoWriter.Add(w)
		// <-r.Context().Done()
		// icoWriter.Remove(w)
	})

	http.HandleFunc("/grid.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(
			"Content-Type",
			fmt.Sprintf("multipart/x-mixed-replace;boundary=%s", BOUNDARY_STRING),
		)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Connection", "close")

		multiWriter.Add(w)
		<-r.Context().Done()
		multiWriter.Remove(w)
	})

	log.Fatal(http.ListenAndServe(":7792", nil))
}
