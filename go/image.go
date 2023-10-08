package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"
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

func MakeTitledPng(pngBuf *bytes.Buffer, title string) (*bytes.Buffer, error) {
	iENDChunkType := []byte{0, 0, 0, 0, 73, 69, 78, 68}
	before, _, _ := bytes.Cut(pngBuf.Bytes(), iENDChunkType)
	imgBuf := new(bytes.Buffer)
	imgBuf.Write(before)
	err := WriteChunk(imgBuf, []byte("Title\x00"+title), "tEXt")
	if err != nil {
		return nil, err
	}
	err = WriteChunk(imgBuf, nil, "IEND")
	if err != nil {
		return nil, err
	}
	return imgBuf, nil
}

func WriteChunk(buf *bytes.Buffer, b []byte, name string) error {
	n := uint32(len(b))
	if int(n) != len(b) {
		return fmt.Errorf("%v chunk is too large: %v", name, strconv.Itoa(len(b)))
	}
	header := [8]byte{}
	binary.BigEndian.PutUint32(header[:4], n)
	header[4] = name[0]
	header[5] = name[1]
	header[6] = name[2]
	header[7] = name[3]
	footer := [4]byte{}
	crc := crc32.NewIEEE()
	crc.Write(header[4:8])
	crc.Write(b)
	binary.BigEndian.PutUint32(footer[:4], crc.Sum32())

	_, err := buf.Write(header[:8])
	if err != nil {
		return err
	}
	_, err = buf.Write(b)
	if err != nil {
		return err
	}
	_, err = buf.Write(footer[:4])
	if err != nil {
		return err
	}
	return nil
}

func (b BaseImage) Download() {
	f, err := os.Create("./public/background.png")
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

type Line struct {
	p0, p1 image.Point
	width  int
}

func (l Line) Slope() int {
	return (l.p1.Y - l.p0.Y) / (l.p1.X - l.p0.X)
}

func (l Line) InBounds(pt image.Point) bool {
	x, y := float64(pt.X), float64(pt.Y)
	x0, y0 := float64(l.p0.X), float64(l.p0.Y)
	x1, y1 := float64(l.p1.X), float64(l.p1.Y)
	t := ((x-x0)*(x1-x0) + (y-y0)*(y1-y0)) / (math.Pow(x1-x0, 2) + math.Pow(y1-y0, 2))
	if t < 0 || t > 1 {
		return false
	}
	d := math.Pow(x-x0-t*(x1-x0), 2) + math.Pow(y-y0-t*(y1-y0), 2)
	return d <= float64(l.width*l.width)/4
}

func (l Line) GetRandPoint() image.Point {
	p0, p1 := l.p0, l.p1
	if p0.X == p1.X {
		return image.Pt(p0.X, RandInRange(p0.Y, p1.Y))
	}
	m := l.Slope()
	x := RandInRange(p0.X, p0.X)
	y := (p1.Y-p0.Y)*m + p0.Y
	return image.Pt(x, y)
}

type Circle struct {
	center image.Point
	radius int
}

func (c Circle) Distance(pt image.Point) float64 {
	diff := pt.Sub(c.center)
	return math.Sqrt(float64(diff.X*diff.X) + float64(diff.Y*diff.Y))
}

func (c Circle) InBounds(pt image.Point) bool {
	return c.Distance(pt) < float64(c.radius)
}

func RandInRange(x, y int) int {
	var min, max int
	if x < y {
		min = x
		max = y
	} else {
		min = y
		max = x
	}
	if min == max {
		return min
	}
	rand.Seed(time.Now().UnixNano())
	if min < 0 && max < 0 {
		return rand.Intn(-min+max) + min
	} else {
		return rand.Intn(max-min) + min
	}
}

func MakeRandPoint(rect image.Rectangle) image.Point {
	x0 := RandInRange(rect.Min.X, rect.Max.X)
	y0 := RandInRange(rect.Min.Y, rect.Max.Y)
	return image.Pt(x0, y0)
}

func MakeRandRect(rect image.Rectangle) image.Rectangle {
	x0 := RandInRange(rect.Min.X, rect.Max.X)
	y0 := RandInRange(rect.Min.Y, rect.Max.Y)
	x1 := RandInRange(rect.Min.X, rect.Max.X)
	y1 := RandInRange(rect.Min.Y, rect.Max.Y)
	return image.Rect(x0, y0, x1, y1).Intersect(rect)
}

func ManhattanDistance(p0, p1 image.Point) int {
	dx, dy := 0, 0
	if p1.X > p0.X {
		dx = p1.X - p0.X
	} else {
		dx = p0.X - p1.X
	}
	if p1.Y > p0.Y {
		dy = p1.Y - p0.Y
	} else {
		dy = p0.Y - p1.Y
	}
	return dx + dy
}

func HuetoRGB(p, q, h float64) uint8 {
	if h < 0 {
		h += 1
	}
	if h > 1 {
		h -= 1
	}
	if h < float64(1)/float64(6) {
		return uint8(math.Round(float64(math.MaxUint8)*p + (q-p)*float64(6)*h))
	}
	if h < float64(1)/float64(2) {
		return uint8(math.Round(float64(math.MaxUint8) * q))
	}
	if h < float64(2)/float64(3) {
		return uint8(math.Round(float64(math.MaxUint8)*p + (q-p)*(float64(2)/float64(3)-h)*float64(6)))
	}
	return uint8(math.Round(float64(math.MaxUint8) * p))
}

func HSLtoRGB(h, s, l float64) (r, g, b uint8) {
	if s == 0 {
		// Achromatic
		r = uint8(math.Round(float64(l) * float64(math.MaxUint8)))
		g = r
		b = r
	} else {
		var q float64
		if l < 0.5 {
			l = 0.5
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := float64(2)*l - q
		r = HuetoRGB(p, q, h+(float64(1)/float64(3)))
		g = HuetoRGB(p, q, h)
		b = HuetoRGB(p, q, h-(float64(1)/float64(3)))
	}
	return
}
