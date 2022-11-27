package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

type BaseImage struct {
	bounds image.Rectangle
}

func (b BaseImage) ColorModel() color.Model {
	return color.RGBAModel
}

func (b BaseImage) Bounds() image.Rectangle {
	return b.bounds
}

func (b BaseImage) At(x, y int) color.Color {
	if x%10 == 0 || y%10 == 0 || float64(x+1) == math.Abs(WORLD_BOUNDS/2) || float64(y+1) == math.Abs(WORLD_BOUNDS/2) {
		return color.Black
	}
	return color.White
}

func (b BaseImage) Download() {
	f, err := os.Create("public/background.png")
	if err != nil {
		log.Fatal(err)
	}

	if err := png.Encode(f, b); err != nil {
		f.Close()
		log.Fatal(err)
	}

	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func MakeBaseImage() BaseImage {
	return BaseImage{
		bounds: image.Rect(-WORLD_BOUNDS/2, -WORLD_BOUNDS/2, WORLD_BOUNDS/2, WORLD_BOUNDS/2),
	}
}
