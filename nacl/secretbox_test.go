package nacl

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"testing"
)

func TestSecretBox(t *testing.T) {
	var secretKey [32]byte
	if _, err := rand.Reader.Read(secretKey[:]); err != nil {
		t.Fatal(err)
	}

	t.Run("happy path", func(t *testing.T) {
		box := NewSecretBox(&secretKey, nil)

		want := []byte("hello world")
		encrypted := mustSealMessage(t, want, box)

		got := mustOpenMessage(t, encrypted, box)

		if string(want) != string(got) {
			t.Fatalf("want msg: %s, got %s", want, got)
		}
	})

	t.Run("bad nonce", func(t *testing.T) {
		box := NewSecretBox(&secretKey, &staticReader{
			err: errors.New("boom"),
		})

		_, err := box.Seal([]byte("hello world"))
		if err != ErrNonceRead {
			t.Fatalf("want error: %v, got %v", ErrNonceRead, err)
		}
	})

	t.Run("bad encrypted message", func(t *testing.T) {
		box := NewSecretBox(&secretKey, nil)

		encrypted := mustSealMessage(t, []byte("hello world"), box)

		// tamper with message
		encrypted[0], encrypted[1] = encrypted[1], encrypted[0]

		if _, err := box.Open(encrypted); err != ErrDecrypt {
			t.Fatalf("want error: %v, got %v", ErrDecrypt, err)
		}
	})

	t.Run("encrypted message too short", func(t *testing.T) {
		box := NewSecretBox(&secretKey, nil)

		// Try to open a 23-length msg (min length is 24)
		if _, err := box.Open([]byte("12345678901234567890123")); err != ErrEncryptedMsgTooShort {
			t.Fatalf("want error: %v, got %v", ErrEncryptedMsgTooShort, err)
		}
	})

	t.Run("get secret key sig", func(t *testing.T) {
		box := NewSecretBox(&secretKey, nil)
		sig := box.GetSecretKeySig()

		xwant := md5.Sum([]byte(hex.EncodeToString(secretKey[:])))
		want := hex.EncodeToString(xwant[:])
		if sig != want {
			t.Fatalf("want signature: %v, got %v", want, sig)
		}
	})
}

func TestMultiSecretBox(t *testing.T) {
	const wantMessage = "hello world"

	newBox := NewSecretBox(mustSecretKey(t), nil)
	newBoxSealed := mustSealMessage(t, []byte(wantMessage), newBox)

	oldBox := NewSecretBox(mustSecretKey(t), nil)
	oldBoxSealed := mustSealMessage(t, []byte(wantMessage), oldBox)

	mbox := NewMultiSecretBox(newBox, oldBox)

	// ensure we can unseal messages from wrapped boxes
	for _, sealed := range [][]byte{newBoxSealed, oldBoxSealed} {
		gotMessage, err := mbox.Open(sealed)
		if err != nil {
			t.Fatal(err)
		}

		if wantMessage != string(gotMessage) {
			t.Fatalf("want message: %s, got %s", wantMessage, gotMessage)
		}
	}

	// ensure new messages use the first box
	multiBoxSealed := mustSealMessage(t, []byte(wantMessage), mbox)
	gotMessage := mustOpenMessage(t, multiBoxSealed, newBox)
	if wantMessage != string(gotMessage) {
		t.Fatalf("want message: %s, got %s", wantMessage, gotMessage)
	}

	if mbox.GetSecretKeySig() != newBox.GetSecretKeySig() {
		t.Fatalf("want signature: %v, got: %v", newBox.GetSecretKeySig(), mbox.GetSecretKeySig())
	}
}

type opener interface {
	Open(encrypted []byte) ([]byte, error)
}

type sealer interface {
	Seal(message []byte) ([]byte, error)
}

func mustOpenMessage(t *testing.T, sealed []byte, opener opener) []byte {
	t.Helper()
	data, err := opener.Open(sealed)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustSealMessage(t *testing.T, msg []byte, sealer sealer) []byte {
	t.Helper()
	data, err := sealer.Seal(msg)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustSecretKey(t *testing.T) *[32]byte {
	t.Helper()
	var secretKey [32]byte
	if _, err := rand.Reader.Read(secretKey[:]); err != nil {
		t.Fatal(err)
	}
	return &secretKey
}

type staticReader struct {
	err error
	n   int
}

func (b *staticReader) Read(_ []byte) (int, error) {
	return b.n, b.err
}
