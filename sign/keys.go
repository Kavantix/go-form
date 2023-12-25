package sign

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
)

var (
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
)

func LoadKeys(privPath, pubPath string) error {

	privKeyBytes, err := os.ReadFile(privPath)
	if err != nil {
		return fmt.Errorf("cannot read private key: %w", err)
	}
	privKeyBytes, err = Base64Decode(privKeyBytes)
	if err != nil {
		return fmt.Errorf("cannot decode private key: %w", err)
	}
	privateKey = ed25519.PrivateKey(privKeyBytes)
	pubKeyBytes, err := os.ReadFile(pubPath)
	if err != nil {
		return fmt.Errorf("cannot read public key: %w", err)
	}
	if err != nil {
		return fmt.Errorf("cannot decode public key: %w", err)
	}
	pubKeyBytes, err = Base64Decode(pubKeyBytes)
	publicKey = ed25519.PublicKey(pubKeyBytes)
	return nil
}

func Base64Decode(bytes []byte) ([]byte, error) {
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(bytes)))
	n, err := base64.StdEncoding.Decode(dst, bytes)
	if err != nil {
		return nil, err
	}
	return dst[:n], nil
}

func Base64Encode(bytes []byte) []byte {
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(bytes)))
	base64.StdEncoding.Encode(dst, bytes)
	return dst
}
