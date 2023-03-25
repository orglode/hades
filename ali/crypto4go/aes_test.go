package crypto4go

import (
	"encoding/hex"
	"testing"
)

// AES Tool https://www.javainuse.com/aesgenerator
// AES Tool https://www.lddgo.net/en/encrypt/aes

func TestAESCBCEncrypt(t *testing.T) {
	var testTbl = []struct {
		plaintext  []byte
		key        []byte
		iv         []byte
		ciphertext string
	}{
		{
			plaintext:  []byte("test data"),
			key:        []byte("test-key-aes-128"),
			iv:         []byte("1111111111111111"),
			ciphertext: "7ec0cf2582f2d197e13add801d22f346",
		},
		{
			plaintext:  []byte("test data"),
			key:        []byte("test-key-aes-192-0000000"),
			iv:         []byte("1111111111111111"),
			ciphertext: "e4222853d29dbfa2cb2c2799d8d1e0ed",
		},
		{
			plaintext:  []byte("test data"),
			key:        []byte("test-key-aes-192-000000000000000"),
			iv:         []byte("1111111111111111"),
			ciphertext: "f279bca14e6fa25cca3439f2e1793358",
		},
	}

	for _, test := range testTbl {
		var ciphertext, err = AESCBCEncrypt(test.plaintext, test.key, test.iv)
		if err != nil {
			t.Fatal(err)
		}

		var r = hex.EncodeToString(ciphertext)

		if r != test.ciphertext {
			t.Fatalf("AES CBC 加密 %s 结果，期望: %s, 实际: %s \n", string(test.plaintext), test.ciphertext, r)
		}
	}
}

func TestAESCBCDecrypt(t *testing.T) {
	var testTbl = []struct {
		plaintext  string
		key        []byte
		iv         []byte
		ciphertext string
	}{
		{
			plaintext:  "test data",
			key:        []byte("test-key-aes-128"),
			iv:         []byte("1111111111111111"),
			ciphertext: "7ec0cf2582f2d197e13add801d22f346",
		},
		{
			plaintext:  "test data",
			key:        []byte("test-key-aes-192-0000000"),
			iv:         []byte("1111111111111111"),
			ciphertext: "e4222853d29dbfa2cb2c2799d8d1e0ed",
		},
		{
			plaintext:  "test data",
			key:        []byte("test-key-aes-192-000000000000000"),
			iv:         []byte("1111111111111111"),
			ciphertext: "f279bca14e6fa25cca3439f2e1793358",
		},
	}

	for _, test := range testTbl {
		var ciphertext, _ = hex.DecodeString(test.ciphertext)

		var plaintext, err = AESCBCDecrypt(ciphertext, test.key, test.iv)
		if err != nil {
			t.Fatal(err)
		}

		var r = string(plaintext)

		if r != test.plaintext {
			t.Fatalf("AES CBC 解密 %s 结果，期望: %s, 实际: %s \n", test.ciphertext, test.plaintext, r)
		}
	}
}

func TestAESGCMDecryptWithNonce(t *testing.T) {
	var testTbl = []struct {
		ciphertext string
		plaintext  string
		nonce      []byte
		key        []byte
	}{
		{
			ciphertext: "b65f128867485ca4dca364218fdadaed7565593e",
			plaintext:  "test",
			nonce:      []byte("123456789111"),
			key:        []byte("test-key-aes-128"),
		},
		{
			ciphertext: "aa5f0d9059d9347efd75fc92cd59346d9afef4c2e3",
			plaintext:  "hello",
			nonce:      []byte("123456789111"),
			key:        []byte("test-key-aes-128"),
		},
	}

	for _, test := range testTbl {
		var ciphertext, _ = hex.DecodeString(test.ciphertext)

		var plaintext, err = AESGCMDecryptWithNonce(ciphertext, test.key, test.nonce, nil)
		if err != nil {
			t.Fatal(err)
		}

		var r = string(plaintext)

		if r != test.plaintext {
			t.Fatalf("AES GCM 解密 %s 结果，期望: %s, 实际: %s \n", test.ciphertext, test.plaintext, r)
		}
	}
}
