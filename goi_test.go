package goi

import (
	"bytes"
	"image/png"
	"os"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	f, err := os.Open("goi.png")
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	wr := bytes.NewBuffer(nil)
	if err := Encode(wr, img); err != nil {
		t.Fatal(err)
	}

	img2, err := Decode(bytes.NewReader(wr.Bytes()))
	if err != nil {
		t.Fatal(err)
	}

	if img.Bounds() != img2.Bounds() {
		t.Fatal("bound mismatch")
	}

	i := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r1, g1, b1, a1 := img.At(x, y).RGBA()
			r2, g2, b2, a2 := img2.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				t.Fatal("At mismatch", i, x, y, img.At(x, y), img2.At(x, y))
			}
			i++
		}
	}
}

func TestEncode(t *testing.T) {
	f, err := os.Open("goi.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	f2, err := os.Open("goi.qoi")
	defer f2.Close()
	if err != nil {
		t.Fatal(err)
	}
	img2, err := Decode(f2)
	if err != nil {
		t.Fatal(err)
	}

	if img.Bounds() != img2.Bounds() {
		t.Fatal("bound mismatch")
	}

	i := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r1, g1, b1, a1 := img.At(x, y).RGBA()
			r2, g2, b2, a2 := img2.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				t.Fatal("At mismatch", i, x, y, img.At(x, y), img2.At(x, y))
			}
			i++
		}
	}
}
