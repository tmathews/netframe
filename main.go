package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"io"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
)

func main() {
	var mode string
	var port int
	var window *sdl.Window
	var renderer *sdl.Renderer

	flag.StringVar(&mode, "mode", "debug", "Mode to initiate with")
	flag.IntVar(&port, "port", 8080, "Port to listen on for HTTP")
	flag.Parse()

	windowSize := sdl.Rect{0, 0, 800, 600}
	window, err := sdl.CreateWindow("NETFRAME", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		windowSize.W, windowSize.H, sdl.WINDOW_SHOWN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create window: %s\n", err)
		panic(err)
	}
	defer window.Destroy()

	if mode != "debug" {
		if bounds, err := sdl.GetDisplayBounds(0); err == nil {
			fmt.Println(bounds)
			windowSize = bounds
			window.SetSize(bounds.W, bounds.H)
			window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
		}
	}

	renderer, err = sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create renderer: %s\n", err)
		panic(err)
	}
	defer renderer.Destroy()

	r := gin.Default()
	r.PUT("/image", func(c *gin.Context) {
		file, err := writeImageFile()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		}
		defer file.Close()
		io.Copy(file, c.Request.Body)
		if err := displayPicture(renderer, file.Name()); err != nil {
			c.String(http.StatusBadRequest, err.Error())
		}
		c.String(http.StatusOK, "")
	})

	if f, err := readImageFile(); err == nil {
		displayPicture(renderer, f.Name())
		f.Close()
	}

	server := http.Server{
		Addr:    "127.0.0.1:" + strconv.Itoa(port),
		Handler: r,
	}
	go func() {
		for {
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch event.(type) {
				case *sdl.QuitEvent:
					server.Close()
					break
				}
			}
		}
	}()
	server.ListenAndServe()
}

func getImageFilename() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return path.Join(u.HomeDir, ".netframe-img"), nil
}

func readImageFile() (*os.File, error) {
	name, err := getImageFilename()
	if err != nil {
		return nil, err
	}
	return os.Open(name)
}

func writeImageFile() (*os.File, error) {
	name, err := getImageFilename()
	if err != nil {
		return nil, err
	}
	return os.OpenFile(name, os.O_CREATE | os.O_WRONLY, os.FileMode(int(0655)))
}

func displayPicture(r *sdl.Renderer, filename string) error {
	i, err := img.Load(filename)
	if err != nil {
		return err
	}
	defer i.Free()

	texture, err := r.CreateTextureFromSurface(i)
	if err != nil {
		return err
	}
	defer texture.Destroy()

	bounds := r.GetViewport()
	src := sdl.Rect{0, 0, i.W, i.H}
	dst := getDstRect(i, bounds.W, bounds.H)

	r.Clear()
	r.SetDrawColor(0, 0, 0, 255)
	r.FillRect(&bounds)
	r.Copy(texture, &src, &dst)
	r.Present()

	return nil
}

// TODO fix scale calculations
func getDstRect(i *sdl.Surface, width, height int32) sdl.Rect {
	var dst sdl.Rect
	dst.W = i.W
	dst.H = i.H
	if dst.W > width {
		dst.W = width
		scale := float64(width) / float64(i.W)
		dst.H = int32(scale * float64(dst.H))
	}
	if dst.H > height {
		dst.H = height
		scale := float64(height) / float64(i.H)
		dst.W = int32(scale * float64(dst.W))
	}
	dst.X = int32((float64(width) * 0.5) - (float64(dst.W) * 0.5))
	dst.Y = int32((float64(height) * 0.5) - (float64(dst.H) * 0.5))
	return dst
}
