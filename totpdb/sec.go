package totpdb

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// DeriveKey generates a key using PBKDF2 with SHA-256.
//
// It takes in the password, salt, and key length as parameters
// and returns the derived key as a byte slice.
//
// Parameters:
// - password: the password used to derive the key ([]byte)
// - salt: the salt used to add additional entropy to the key ([]byte)
// - keyLen: the length of the derived key (int)
//
// Returns:
// - []byte: the derived key
func DeriveKey(password, salt []byte, keyLen int) []byte {
	// Use PBKDF2 to derive the key from the password, salt, and key length
	return pbkdf2.Key(password, salt, 4096, keyLen, sha256.New)
}

// GenerateSalt generates a 32-byte salt from a given string.
func GenerateSalt(input string) []byte {
	hash := sha256.Sum256([]byte(input))
	return hash[:] //??
}

// Encrypt encrypts the plaintext using AES-GCM.
//
// It takes in the plaintext and key as parameters and returns the encrypted
// ciphertext and an error if any occurred.
//
// Parameters:
// - src: the plaintext to be encrypted ([]byte)
// - key: the key used to encrypt the plaintext ([]byte)
//
// Returns:
// - []byte: the encrypted ciphertext
// - error: an error if any occurred during encryption
func Encrypt(src, key []byte) ([]byte, error) {
	// Create a new AES cipher block using the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create a new AES-GCM cipher mode using the cipher block
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate a random nonce of the appropriate size for the cipher mode
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt the plaintext using the cipher mode, nonce, and additional data
	ciphertext := aesGCM.Seal(nonce, nonce, src, nil)
	return ciphertext, nil
}

// Decrypt decrypts the ciphertext using AES-GCM.
//
// Takes in the ciphertext and key as parameters and returns the decrypted
// plaintext and an error if any occurred.
//
// Parameters:
// - src: the ciphertext to be decrypted ([]byte)
// - key: the key used to decrypt the ciphertext ([]byte)
//
// Returns:
// - []byte: the decrypted plaintext
// - error: an error if any occurred during decryption
func Decrypt(src, key []byte) ([]byte, error) {
	// Create a new AES cipher block using the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create a new AES-GCM cipher mode using the cipher block
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Get the nonce size
	nonceSize := aesGCM.NonceSize()

	// Check if the ciphertext is too short to contain the nonce
	if len(src) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Split the ciphertext into the nonce and the actual ciphertext
	nonce, ciphertext := src[:nonceSize], src[nonceSize:]

	// Decrypt the ciphertext using the cipher mode, nonce, and additional data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
