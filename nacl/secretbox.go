package nacl

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

var (
	// ErrNonceRead is returned when the provided reader fails to generate a nonce.
	ErrNonceRead = errors.New("failed to read nonce value")

	// ErrDecrypt is returned when opening an encrypted message fails.
	ErrDecrypt = errors.New("failed to decrypt value")

	// ErrEncryptedMsgTooShort is returned when the encrypted message is shorter than 24 bytes
	ErrEncryptedMsgTooShort = errors.New("failed to read encrypted message: minimum length = 24")
)

// Box opens and seals secrets.
type Box interface {
	Seal(message []byte) ([]byte, error)
	Open(encrypted []byte) ([]byte, error)
	GetSecretKeySig() string
}

var _ Box = &SecretBox{}

// NewSecretBox creates a new SecretBox with the provided secret key and optional nonce reader.
func NewSecretBox(key *[32]byte, nonceReader io.Reader) *SecretBox {
	if nonceReader == nil {
		nonceReader = rand.Reader
	}

	return &SecretBox{
		key:         key,
		nonceReader: nonceReader,
	}
}

// SecretBox provides nacl secretbox encryption.
type SecretBox struct {
	key         *[32]byte
	nonceReader io.Reader
}

// Seal encrypts the provided message.
func (s *SecretBox) Seal(message []byte) ([]byte, error) {
	nonce, err := s.nonce()
	if err != nil {
		return nil, ErrNonceRead
	}
	return secretbox.Seal(nonce[:], message, nonce, s.key), nil
}

// Open decrypts the provided message.
func (s *SecretBox) Open(encrypted []byte) ([]byte, error) {
	if len(encrypted) < 24 {
		return nil, ErrEncryptedMsgTooShort
	}

	var decryptNonce [24]byte
	copy(decryptNonce[:], encrypted[:24])

	decrypted, ok := secretbox.Open(nil, encrypted[24:], &decryptNonce, s.key)
	if !ok {
		return nil, ErrDecrypt
	}
	return decrypted, nil
}

// GetSecretKeySig returns the md5 hash sum of the key
func (s *SecretBox) GetSecretKeySig() string {
	sig := md5.Sum([]byte(hex.EncodeToString((*s.key)[:])))
	return hex.EncodeToString(sig[:])
}

func (s *SecretBox) nonce() (*[24]byte, error) {
	var nonce [24]byte
	if _, err := io.ReadFull(s.nonceReader, nonce[:]); err != nil {
		return nil, err
	}
	return &nonce, nil
}

var _ Box = &MultiSecretBox{}

// NewMultiSecretBox creates a MultiSecretBox which wraps the provided SecretBox.
func NewMultiSecretBox(boxes ...*SecretBox) *MultiSecretBox {
	return &MultiSecretBox{boxes}
}

// MultiSecretBox wraps multiple SecretBox.
type MultiSecretBox struct {
	boxes []*SecretBox
}

// Seal encrypts the provided message with the first SecretBox.
func (m *MultiSecretBox) Seal(message []byte) ([]byte, error) {
	return m.boxes[0].Seal(message)
}

// Open attempts to decrypt the provided message with all available SecretBox.
func (m *MultiSecretBox) Open(encrypted []byte) (message []byte, err error) {
	for _, box := range m.boxes {
		message, err = box.Open(encrypted)
		if err == nil {
			break
		}
	}
	return
}

// GetSecretKeySig returns the md5 hash sum of the key
func (m *MultiSecretBox) GetSecretKeySig() string {
	return m.boxes[0].GetSecretKeySig()
}
