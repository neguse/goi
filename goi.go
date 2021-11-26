package goi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
)

const (
	qoiIndex  = 0x00 // 00xxxxxx
	qoiRun8   = 0x40 // 010xxxxx
	qoiRun16  = 0x60 // 011xxxxx
	qoiDiff8  = 0x80 // 10xxxxxx
	qoiDiff16 = 0xc0 // 110xxxxx
	qoiDiff24 = 0xe0 // 1110xxxx
	qoiColor  = 0xf0 // 1111xxxx
	qoiMask2  = 0xc0 // 11000000
	qoiMask3  = 0xe0 // 11100000
	qoiMask4  = 0xf0 // 11110000
)

func colorHash(c color.RGBA) byte {
	return c.R ^ c.G ^ c.B ^ c.A
}

func rgbaColor(c color.Color) color.RGBA {
	if rgba, ok := c.(color.RGBA); ok {
		return rgba
	} else if nrgba, ok := c.(color.NRGBA); ok {
		return color.RGBA{
			R: nrgba.R,
			G: nrgba.G,
			B: nrgba.B,
			A: nrgba.A,
		}
	}
	r, g, b, a := c.RGBA()
	return color.RGBA{
		R: uint8(r << 8),
		G: uint8(g << 8),
		B: uint8(b << 8),
		A: uint8(a << 8),
	}
}

const (
	magic      = "qoif"
	headerSize = 12
	padding    = 4
)

type errWriter struct {
	err error
	wr  io.Writer
}

func (w *errWriter) Write(b byte) {
	if w.err != nil {
		return
	}
	w.err = binary.Write(w.wr, binary.BigEndian, b)
}

func cond3(cond int, then int, els int) int {
	if cond != 0 {
		return then
	}
	return els
}

func Encode(w io.Writer, m image.Image) error {
	data := bytes.NewBuffer(nil)
	wr := errWriter{wr: data}

	var index [64]color.RGBA
	run := 0
	pxPrev := color.RGBA{A: 255}

	for y := m.Bounds().Min.Y; y < m.Bounds().Max.Y; y++ {
		for x := m.Bounds().Min.X; x < m.Bounds().Max.X; x++ {
			px := rgbaColor(m.At(x, y))
			if px == pxPrev {
				run++
			}

			last := (x == m.Bounds().Max.X-1) && (y == m.Bounds().Max.Y-1)
			if run > 0 && (run == 0x2020 || px != pxPrev || last) {
				if run < 33 {
					run -= 1
					wr.Write(byte(qoiRun8 | run))
				} else {
					run -= 33
					wr.Write(byte(qoiRun16 | run>>8))
					wr.Write(byte(run))
				}
				run = 0
			}

			if px != pxPrev {
				indexPos := colorHash(px) % 64

				if index[indexPos] == px {
					wr.Write(qoiIndex | indexPos)
				} else {
					index[indexPos] = px

					var vr int = int(px.R) - int(pxPrev.R)
					var vg int = int(px.G) - int(pxPrev.G)
					var vb int = int(px.B) - int(pxPrev.B)
					var va int = int(px.A) - int(pxPrev.A)

					if vr > -16 && vr < 17 && vg > -16 && vg < 17 &&
						vb > -16 && vb < 17 && va > -16 && va < 17 {
						if va == 0 && vr > -2 && vr < 3 &&
							vg > -2 && vg < 3 && vb > -2 && vb < 3 {
							wr.Write(byte(qoiDiff8 | ((vr + 1) << 4) | (vg+1)<<2 | (vb + 1)))
						} else if va == 0 && vr > -16 && vr < 17 &&
							vg > -8 && vg < 9 && vb > -8 && vb < 9 {
							wr.Write(byte(qoiDiff16 | (vr + 15)))
							wr.Write(byte(((vg + 7) << 4) | (vb + 7)))
						} else {
							wr.Write(byte(qoiDiff24 | ((vr + 15) >> 1)))
							wr.Write(byte(((vr + 15) << 7) | ((vg + 15) << 2) | ((vb + 15) >> 3)))
							wr.Write(byte(((vb + 15) << 5) | (va + 15)))
						}
					} else {
						wr.Write(byte(qoiColor | (cond3(vr, 8, 0)) | (cond3(vg, 4, 0)) | (cond3(vb, 2, 0)) | (cond3(va, 1, 0))))
						if vr != 0 {
							wr.Write(px.R)
						}
						if vg != 0 {
							wr.Write(px.G)
						}
						if vb != 0 {
							wr.Write(px.B)
						}
						if va != 0 {
							wr.Write(px.A)
						}
					}
				}
			}
			pxPrev = px
		}
	}

	for i := 0; i < padding; i++ {
		wr.Write(0)
	}

	dataLen := len(data.Bytes())

	if err := binary.Write(w, binary.BigEndian, []byte(magic)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint16(m.Bounds().Dx())); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint16(m.Bounds().Dy())); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint32(dataLen)); err != nil {
		return err
	}
	if _, err := io.Copy(w, bytes.NewReader(data.Bytes())); err != nil {
		return err
	}

	return nil
}

var errInvalidHeader = errors.New("invalid header")
var errInvalidFileSize = errors.New("invalid file size")

func Decode(r io.Reader) (image.Image, error) {
	mgc := make([]byte, 4)
	if err := binary.Read(r, binary.BigEndian, &mgc); err != nil {
		return nil, err
	}
	var w, h uint16
	if err := binary.Read(r, binary.BigEndian, &w); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &h); err != nil {
		return nil, err
	}

	var dataLen uint32
	if err := binary.Read(r, binary.BigEndian, &dataLen); err != nil {
		return nil, err
	}

	if !bytes.Equal(mgc, []byte(magic)) {
		return nil, errInvalidHeader
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if int(dataLen) != len(data) {
		return nil, errInvalidFileSize
	}

	m := image.NewRGBA(image.Rectangle{
		Max: image.Point{X: int(w), Y: int(h)},
	})

	pxLen := int(w) * int(h) * 4
	px := color.RGBA{A: 255}
	var index [64]color.RGBA

	p := 0
	run := 0
	chunksLen := dataLen - padding

	for pxPos := 0; pxPos < int(pxLen); pxPos += 4 {
		if run > 0 {
			run--
		} else if p < int(chunksLen) {
			b1 := data[p]
			p++

			if (b1 & qoiMask2) == qoiIndex {
				px = index[b1^qoiIndex]
			} else if (b1 & qoiMask3) == qoiRun8 {
				run = int(b1 & 0x1f)
			} else if (b1 & qoiMask3) == qoiRun16 {
				b2 := int(data[p])
				p++
				run = ((int(b1&0x1f) << 8) | (b2)) + 32
			} else if (b1 & qoiMask2) == qoiDiff8 {
				px.R += ((b1 >> 4) & 0x03) - 1
				px.G += ((b1 >> 2) & 0x03) - 1
				px.B += (b1 & 0x03) - 1
			} else if (b1 & qoiMask3) == qoiDiff16 {
				b2 := int(data[p])
				p++
				px.R += byte(b1&0x1f) - 15
				px.G += byte(b2>>4) - 7
				px.B += byte(b2&0x0f) - 7
			} else if (b1 & qoiMask4) == qoiDiff24 {
				b2 := int(data[p])
				p++
				b3 := int(data[p])
				p++
				px.R += byte((int(b1&0x0f)<<1)|(b2>>7)) - 15
				px.G += byte((b2&0x7c)>>2) - 15
				px.B += byte(((b2&0x03)<<3)|((b3&0xe0)>>5)) - 15
				px.A += byte(b3&0x1f) - 15
			} else if (b1 & qoiMask4) == qoiColor {
				if b1&8 != 0 {
					px.R = data[p]
					p++
				}
				if b1&4 != 0 {
					px.G = data[p]
					p++
				}
				if b1&2 != 0 {
					px.B = data[p]
					p++
				}
				if b1&1 != 0 {
					px.A = data[p]
					p++
				}
			}

			index[colorHash(px)%64] = px
		}

		m.Pix[pxPos], m.Pix[pxPos+1], m.Pix[pxPos+2], m.Pix[pxPos+3] = px.R, px.G, px.B, px.A
	}
	return m, nil
}
