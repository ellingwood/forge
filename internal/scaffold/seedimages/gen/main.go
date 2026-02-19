// gen generates the seed hero PNG images used by forge new site --seed.
// Run from the seedimages/ directory: go run gen/main.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	// Write output next to this file's parent directory (i.e., seedimages/).
	_, file, _, _ := runtime.Caller(0)
	outDir := filepath.Dir(filepath.Dir(file))

	images := []struct {
		name string
		c    color.RGBA
	}{
		{"hero-go.png", color.RGBA{R: 0x29, G: 0x63, B: 0xD0, A: 0xFF}},       // blue
		{"hero-static-sites.png", color.RGBA{R: 0x4F, G: 0x46, B: 0xE5, A: 0xFF}}, // indigo
		{"hero-cloud.png", color.RGBA{R: 0x0D, G: 0x94, B: 0x88, A: 0xFF}},    // teal
	}

	for _, img := range images {
		m := image.NewRGBA(image.Rect(0, 0, 800, 300))
		for y := range 300 {
			for x := range 800 {
				m.Set(x, y, img.c)
			}
		}
		path := filepath.Join(outDir, img.name)
		f, err := os.Create(path)
		if err != nil {
			panic(err)
		}
		if err := png.Encode(f, m); err != nil {
			panic(err)
		}
		f.Close()
	}
}
