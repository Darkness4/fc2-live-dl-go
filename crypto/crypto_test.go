package crypto_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/crypto"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	pt := make([]byte, 1024)
	_, err := rand.Read(pt)
	require.NoError(t, err)

	key := make([]byte, 32)
	_, err = rand.Read(key)
	require.NoError(t, err)

	var w bytes.Buffer
	t.Run("Encrypt", func(t *testing.T) {
		err := crypto.Encrypt(&w, key, []byte(pt))
		require.NoError(t, err)
	})

	t.Run("Decrypt", func(t *testing.T) {
		actual, err := crypto.Decrypt(&w, key)
		require.NoError(t, err)
		require.Equal(t, pt, actual)
	})
}
