package cryptoRandom

import (
	"crypto/rand"
)

const encodeStr = "ABCDEFGHJKLMNPQRSTUVWXYZ" +
	"abcdefghijkmnpqrstuvwxyz" +
	"0123456789" +
	"-_+=,."
const mask63 = uint8(len(encodeStr) - 1)

func AsciiBytes(n int) ([]byte, error) {
	out := make([]byte, n)
	rnd := make([]byte, n)

	if _, err := rand.Read(rnd); err != nil {
		return nil, err
	}

	for i := 0; i < n; i++ {
		pos := uint8(rnd[i]) & mask63
		out[i] = encodeStr[pos]
	}

	return out, nil
}

func AsciiString(n int) (string, error) {
	b, err := AsciiBytes(n)
	if err != nil {
		return "", err
	}
	return string(b), nil
}