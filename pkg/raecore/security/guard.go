package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

var (
	ErrRestrictedOutsideWorking = errors.New("ISO-27000 VIOLATION: RESTRICTED information class is strictly confined to the 'working' layer")
	ErrRestrictedMeshEgress     = errors.New("ISO-27000 VIOLATION: RESTRICTED or Working layer memory cannot be exported over RAE-Mesh")
	ErrAbsolutePathViolation    = errors.New("C6 CONSTITUTIONAL VIOLATION: absolute filesystem paths and traversal segments are prohibited")
	ErrGracePeriodActive        = errors.New("RAE-CRL EPISTEMIC QUARANTINE: private trace is protected by active 24h Grace Period")
)

// HardFramesGuard enforces ISO 42001/27000 security boundaries, C6 path rules, and cryptographic signing.
type HardFramesGuard struct {
	aesKey    [32]byte
	hmacSecret []byte
}

// NewHardFramesGuard creates a HardFramesGuard derived from the provided secret passphrase.
func NewHardFramesGuard(secret string) *HardFramesGuard {
	if secret == "" {
		secret = "rae-suite-default-sovereign-key-v4"
	}
	key := sha256.Sum256([]byte(secret))
	return &HardFramesGuard{
		aesKey:     key,
		hmacSecret: key[:],
	}
}

// ValidateAndPrepareMemory enforces ISO 27000 INFO_CLASS rules and encrypts RESTRICTED working memories.
func (g *HardFramesGuard) ValidateAndPrepareMemory(item *models.MemoryItem) error {
	item.Layer = models.NormalizeLayer(string(item.Layer))
	if item.InfoClass == "" {
		item.InfoClass = models.InfoInternal
	}
	if item.InfoClass == models.InfoRestricted && item.Layer != models.LayerWorking {
		return ErrRestrictedOutsideWorking
	}
	if item.Layer == models.LayerWorking && item.InfoClass == models.InfoRestricted && !item.IsEncrypted && item.Content != "" {
		enc, err := g.EncryptAESGCM([]byte(item.Content))
		if err != nil {
			return err
		}
		item.EncryptedBytes = enc
		item.IsEncrypted = true
	}
	sum := sha256.Sum256([]byte(item.Content + "|" + string(item.Layer) + "|" + item.TenantID))
	item.ContentHash = hex.EncodeToString(sum[:])
	return nil
}

// ValidateMeshTransfer checks if a memory item is eligible for RAE-Mesh / CMT federation.
func (g *HardFramesGuard) ValidateMeshTransfer(item models.MemoryItem) error {
	if item.InfoClass == models.InfoRestricted || models.NormalizeLayer(string(item.Layer)) == models.LayerWorking {
		return ErrRestrictedMeshEgress
	}
	return nil
}

// ValidateEpistemicMeshTransfer checks if an RAE-CRL artifact can be synced over RAE-Mesh.
func (g *HardFramesGuard) ValidateEpistemicMeshTransfer(art models.EpistemicArtifact, now time.Time) error {
	if art.IsInGracePeriod(now) {
		return ErrGracePeriodActive
	}
	if art.InfoClass == models.InfoRestricted || art.Visibility == models.VisibilityPrivate {
		return ErrRestrictedMeshEgress
	}
	return nil
}

// ValidateProjectRelativePath enforces Constitutional Rule C6 (no absolute paths or '..' traversal).
func (g *HardFramesGuard) ValidateProjectRelativePath(p string) error {
	trimmed := strings.TrimSpace(p)
	if strings.HasPrefix(trimmed, "/") || strings.Contains(trimmed, "..") || strings.Contains(trimmed, ":\\") {
		return fmt.Errorf("%w: %s", ErrAbsolutePathViolation, p)
	}
	return nil
}

// EncryptAESGCM encrypts plaintext using AES-256-GCM.
func (g *HardFramesGuard) EncryptAESGCM(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(g.aesKey[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptAESGCM decrypts ciphertext produced by EncryptAESGCM.
func (g *HardFramesGuard) DecryptAESGCM(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(g.aesKey[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(ciphertext) < ns {
		return nil, errors.New("ciphertext too short")
	}
	nonce, enc := ciphertext[:ns], ciphertext[ns:]
	return gcm.Open(nil, nonce, enc, nil)
}

// SignTokenHMAC generates a deterministic HMAC-SHA256 hex signature over a payload string.
func (g *HardFramesGuard) SignTokenHMAC(payload string) string {
	mac := hmac.New(sha256.New, g.hmacSecret)
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyTokenHMAC verifies an HMAC-SHA256 hex signature in constant time.
func (g *HardFramesGuard) VerifyTokenHMAC(payload, signatureHex string) bool {
	expected := g.SignTokenHMAC(payload)
	return hmac.Equal([]byte(expected), []byte(signatureHex))
}
