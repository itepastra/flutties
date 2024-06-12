package helpers

import (
	"encoding/hex"
)

func PxToHex(color uint32) string {
	cbytes := []byte{byte(color >> 24), byte(color >> 16), byte(color >> 8)}
	return hex.EncodeToString(cbytes)
}
