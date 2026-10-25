package bun

import (
	"github.com/danfragoso/thdwb/assets"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var sansSerif = map[int]font.Face{
	300: parseFont(300),
	400: parseFont(400),
	600: parseFont(600),
	700: parseFont(700),
	800: parseFont(800),
}

func parseFont(weight int) font.Face {
	fontBytes := assets.OpenSans(weight)
	f, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    20,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil
	}
	return face
}
