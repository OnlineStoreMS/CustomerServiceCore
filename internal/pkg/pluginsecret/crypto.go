package pluginsecret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

type Codec struct {
	gcm cipher.AEAD
}

func NewCodec(keyMaterial string) (*Codec, error) {
	keyMaterial = strings.TrimSpace(keyMaterial)
	if keyMaterial == "" {
		return nil, fmt.Errorf("plugin secret key is empty")
	}
	sum := sha256.Sum256([]byte(keyMaterial))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Codec{gcm: gcm}, nil
}

func (c *Codec) Encrypt(plain string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("plugin secret codec is nil")
	}
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := c.gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.RawURLEncoding.EncodeToString(out), nil
}

func (c *Codec) Decrypt(enc string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("plugin secret codec is nil")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(enc))
	if err != nil {
		return "", err
	}
	ns := c.gcm.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	plain, err := c.gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
