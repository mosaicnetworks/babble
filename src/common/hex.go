package common

import (
	"encoding/hex"
	"fmt"
)

//EncodeToString returns the UPPERCASE string representation of hexBytes with
//the 0X prefix
func EncodeToString(hexBytes []byte) string {
	return fmt.Sprintf("0X%X", hexBytes)
}

//DecodeFromString converts a hex string with 0X prefix to a byte slice
func DecodeFromString(hexString string) ([]byte, error) {
	return hex.DecodeString(hexString[2:])
}
