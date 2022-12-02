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
	"os"
	"strconv"
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
		return fmt.Errorf("%v chunk is too large: ", name, strconv.Itoa(len(b)))
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
