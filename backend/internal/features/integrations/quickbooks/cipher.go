package quickbooks

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

type AESGCMCipher struct {
	aead cipher.AEAD
}

func NewAESGCMCipher(encodedKey string) (*AESGCMCipher, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("decode QuickBooks token encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("QuickBooks token encryption key must decode to exactly 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create QuickBooks token cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create QuickBooks token AEAD: %w", err)
	}
	return &AESGCMCipher{aead: aead}, nil
}

func (c *AESGCMCipher) Encrypt(plaintext string) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("create token nonce: %w", err)
	}
	return c.aead.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

func (c *AESGCMCipher) Decrypt(ciphertext []byte) (string, error) {
	nonceSize := c.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("token ciphertext is invalid")
	}
	plaintext, err := c.aead.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt token: %w", err)
	}
	return string(plaintext), nil
}
