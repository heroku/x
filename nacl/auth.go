package nacl

import "golang.org/x/crypto/nacl/auth"

// Signer allows you to sign and verify data.
type Signer interface {
	Sign(data []byte) []byte
	Verify(data, sum []byte) bool
}

type signer struct {
	key *[32]byte
}

// NewSigner creates a single signer with the given key.
func NewSigner(key *[32]byte) Signer {
	return &signer{key: key}
}

// Sign generates a checksum on the given data. It can be verified later via
// Verify.
func (s *signer) Sign(data []byte) []byte {
	res := auth.Sum(data, s.key)
	return res[:]
}

// Verify allows you to verify a previous checksum generated via Sign.
func (s *signer) Verify(sum, data []byte) bool {
	return auth.Verify(sum, data, s.key)
}

// NewMultiSigner allows you to create multiple signers which is useful for
// doing cred rolls.
func NewMultiSigner(signers ...Signer) Signer {
	if len(signers) == 0 {
		panic("multi signer: needs at least one signer")
	}
	return &multiSigner{signers: signers}
}

type multiSigner struct {
	signers []Signer
}

// Sign uses the first signer available to sign.
func (s *multiSigner) Sign(data []byte) []byte {
	return s.signers[0].Sign(data)
}

// Verify uses all available signers until it hits one successful
// verification.
func (s *multiSigner) Verify(sum, data []byte) bool {
	for _, s := range s.signers {
		if s.Verify(sum, data) {
			return true
		}
	}
	return false
}
