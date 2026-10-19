package raemesh

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// DeriveKeyPBKDF2 derives a cryptographic key of keyLen bytes using RFC 2898 PBKDF2-HMAC-SHA256 in pure Go stdlib.
func DeriveKeyPBKDF2(password, salt []byte, iterations, keyLen int) []byte {
	if iterations <= 0 {
		iterations = 10000
	}
	if keyLen <= 0 {
		keyLen = 32
	}
	prf := func(key, data []byte) []byte {
		mac := hmac.New(sha256.New, key)
		_, _ = mac.Write(data)
		return mac.Sum(nil)
	}
	hashLen := 32
	numBlocks := (keyLen + hashLen - 1) / hashLen
	var dk []byte
	for block := 1; block <= numBlocks; block++ {
		var blockBuf [4]byte
		binary.BigEndian.PutUint32(blockBuf[:], uint32(block))
		u := prf(password, append(append([]byte{}, salt...), blockBuf[:]...))
		t := make([]byte, len(u))
		copy(t, u)
		for i := 2; i <= iterations; i++ {
			u = prf(password, u)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		dk = append(dk, t...)
	}
	return dk[:keyLen]
}

// E2EEncryption provides End-to-End AES-256-GCM encryption for RAE-Mesh memory synchronization
// matching Python's rae_core/sync/encryption.py.
type E2EEncryption struct {
	key []byte
}

// NewE2EEncryption initializes AES-256-GCM encryption from a password or raw 32-byte key.
func NewE2EEncryption(password string) *E2EEncryption {
	if password == "" {
		password = "rae-mesh-default-e2e-secret"
	}
	salt := []byte("rae-mesh-v4.6-pbkdf2-salt")
	key := DeriveKeyPBKDF2([]byte(password), salt, 4096, 32)
	return &E2EEncryption{key: key}
}

// Encrypt encrypts plaintext with AES-256-GCM and returns base64(nonce || ciphertext || tag).
func (e *E2EEncryption) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts base64(nonce || ciphertext || tag) produced by Encrypt.
func (e *E2EEncryption) Decrypt(ciphertextB64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// EncryptMemory encrypts the Content field of a MemoryItem for E2E Mesh transit.
func (e *E2EEncryption) EncryptMemory(m models.MemoryItem) (models.MemoryItem, error) {
	ct, err := e.Encrypt(m.Content)
	if err != nil {
		return m, err
	}
	m.Content = ct
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	m.Metadata["e2e_encrypted"] = true
	m.Metadata["cipher"] = "AES-256-GCM-PBKDF2"
	return m, nil
}

// DecryptMemory decrypts the Content field of an E2E-encrypted MemoryItem.
func (e *E2EEncryption) DecryptMemory(m models.MemoryItem) (models.MemoryItem, error) {
	pt, err := e.Decrypt(m.Content)
	if err != nil {
		return m, err
	}
	m.Content = pt
	if m.Metadata != nil {
		m.Metadata["e2e_encrypted"] = false
	}
	return m, nil
}

// EncryptBatch encrypts a batch of MemoryItems for E2E Mesh synchronization.
func (e *E2EEncryption) EncryptBatch(items []models.MemoryItem) ([]models.MemoryItem, error) {
	out := make([]models.MemoryItem, 0, len(items))
	for _, it := range items {
		enc, err := e.EncryptMemory(it)
		if err != nil {
			return nil, err
		}
		out = append(out, enc)
	}
	return out, nil
}

// DecryptBatch decrypts a batch of E2E-encrypted MemoryItems.
func (e *E2EEncryption) DecryptBatch(items []models.MemoryItem) ([]models.MemoryItem, error) {
	out := make([]models.MemoryItem, 0, len(items))
	for _, it := range items {
		dec, err := e.DecryptMemory(it)
		if err != nil {
			return nil, err
		}
		out = append(out, dec)
	}
	return out, nil
}
