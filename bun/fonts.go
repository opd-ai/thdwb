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

// getFontFace returns the font face for the given weight, falling back to 400
// if the weight is not present in the map (prevents nil face dereference).
func getFontFace(weight int) font.Face {
	face := sansSerif[weight]
	if face == nil {
		face = sansSerif[400]
	}
	return face
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
