package ebitengame

import (
	"log"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

type Faces struct {
	Small  font.Face
	Normal font.Face
	Large  font.Face
}

func loadFaces() Faces {
	fnt, err := truetype.Parse(goregular.TTF)
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}
	mk := func(size float64) font.Face {
		return truetype.NewFace(fnt, &truetype.Options{Size: size, DPI: 72})
	}
	return Faces{
		Small:  mk(12),
		Normal: mk(14),
		Large:  mk(18),
	}
}
