package helpers

import (
	"encoding/hex"
	"image/color"
)

func PxToHex(color color.Color) string {
	r, g, b, _ := color.RGBA()
	cbytes := []byte{byte(r), byte(g), byte(b)}
	return hex.EncodeToString(cbytes)
}
