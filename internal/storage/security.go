package storage

import (
	"encoding/base64"
	"golang.org/x/crypto/argon2"
)

// GetHash - Return the (b64) hash of a key (likely the security key to log in). Username is the salt.
func GetHash(key string, username string) string {
	salt := []byte(username)
	hash := argon2.IDKey([]byte(key), salt, 1, 32*1024, 2, 32)
	return base64.StdEncoding.EncodeToString(hash)
}
