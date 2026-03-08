package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword hashes a plain-text password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword compares a plain-text password against a bcrypt hash.
func CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// HashSHA256 returns the hex-encoded SHA-256 digest of data.
func HashSHA256(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

// GenerateSecureToken returns a cryptographically-random hex string of the
// given byte length (the resulting string will be length*2 characters long).
func GenerateSecureToken(length int) (string, error) {
	b, err := GenerateRandomBytes(length)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateRandomBytes returns n cryptographically-random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
