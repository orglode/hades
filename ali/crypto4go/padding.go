package crypto4go

import (
	"bytes"
	"crypto/aes"
	"errors"
)

var (
	ErrInvalidPadding = errors.New("invalid padding")
)

type Padding func(src []byte, blockSize int) []byte

type UnPadding func(src []byte) []byte

func NoPadding(src []byte, blockSize int) []byte {
	return src
}

func NoUnPadding(src []byte) ([]byte, error) {
	return src, nil
}

func ZeroPadding(src []byte, blockSize int) []byte {
	var pSize = blockSize - len(src)%blockSize
	if pSize == 0 {
		return src
	}
	var pText = bytes.Repeat([]byte{0}, pSize)
	return append(src, pText...)
}

func ZeroUnPadding(src []byte) ([]byte, error) {
	return bytes.TrimFunc(src,
		func(r rune) bool {
			return r == rune(0)
		}), nil
}

func PKCS7Padding(src []byte, blockSize int) []byte {
	var padding = blockSize - len(src)%blockSize
	var pText = bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, pText...)
}

func PKCS7UnPadding(src []byte) ([]byte, error) {
	length := len(src)
	unPadding := int(src[length-1])

	if unPadding > aes.BlockSize || unPadding == 0 {
		return nil, ErrInvalidPadding
	}

	pad := src[len(src)-unPadding:]
	for i := 0; i < unPadding; i++ {
		if pad[i] != byte(unPadding) {
			return nil, ErrInvalidPadding
		}
	}

	return src[:(length - unPadding)], nil
}
