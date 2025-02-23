package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const saltSize = 16

// DeriveKey derives a 32-byte AES key from the secret key using PBKDF2.
func deriveKey(secret []byte) []byte {
	// PBKDF2 is used to derive a key from the secret key
	salt := make([]byte, saltSize) // You can use a random salt in production
	return pbkdf2.Key(secret, salt, 100000, 32, sha256.New)
}

// Encrypt creates a new EncryptWriter.
func Encrypt(w io.Writer, secret []byte, plaintext []byte) error {
	// Derive the key from the secret
	key := deriveKey(secret)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("cannot create cipher: %v", err)
	}

	// Create GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("cannot create GCM cipher: %v", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("cannot generate nonce: %v", err)
	}

	// Storing the nonce in the ciphertext since we have no storage.
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	_, err = w.Write(ciphertext)
	return err
}

// Decrypt reads the encrypted data from the reader and returns the decrypted data.
func Decrypt(r io.Reader, secret []byte) ([]byte, error) {
	// Derive the key from the secret
	key := deriveKey(secret)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cannot create AES cipher: %v", err)
	}

	// Create GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cannot create GCM cipher: %v", err)
	}

	// Read the nonce from the reader (it will be the first part of the encrypted data)
	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(r, nonce)
	if err != nil {
		return nil, fmt.Errorf("cannot read nonce: %v", err)
	}

	// Read the ciphertext from the reader
	ciphertext, err := io.ReadAll(r)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("cannot read ciphertext: %v", err)
	}

	// Decrypt the data
	plainText, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot decrypt data: %v", err)
	}

	return plainText, nil
}
