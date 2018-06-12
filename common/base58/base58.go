package base58

import (
	"crypto/sha256"
	"github.com/shengdoushi/base58"
	"log"
)

var tronAlphabet = base58.NewAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func Encode(input []byte) string {
	return base58.Encode(input, tronAlphabet)
}

func EncodeCheck(input []byte) string {
	h256h0 := sha256.New()
	h256h0.Write(input)
	h0 := h256h0.Sum(nil)

	h256h1 := sha256.New()
	h256h1.Write(h0)
	h1 := h256h1.Sum(nil)

	inputCheck := input
	inputCheck = append(inputCheck, h1[:4]...)

	return Encode(inputCheck)
}

func Decode(input string) ([]byte, error) {
	return base58.Decode(input, tronAlphabet)
}

func DecodeCheck(input string) []byte {
	decodeCheck, err := Decode(input)

	if err != nil {
		log.Fatalln(err.Error())
	}

	if len(decodeCheck) < 4 {
		return nil
	}

	decodeData := decodeCheck[:len(decodeCheck)-4]

	h256h0 := sha256.New()
	h256h0.Write(decodeData)
	h0 := h256h0.Sum(nil)

	h256h1 := sha256.New()
	h256h1.Write(h0)
	h1 := h256h1.Sum(nil)

	if h1[0] == decodeCheck[len(decodeData)] &&
		h1[1] == decodeCheck[len(decodeData)+1] &&
		h1[2] == decodeCheck[len(decodeData)+2] &&
		h1[3] == decodeCheck[len(decodeData)+3] {
		return decodeData
	}

	return nil
}
