// Package hashing provides SHA-256 helpers byte-parity with the bash kernel's
// lib_context.sh:91-113 (context_sha256_string / context_sha256_file), which
// portably shell out to `shasum -a 256` / `sha256sum` and print the lowercase
// hex digest. crypto/sha256 + lowercase-hex encoding reproduces that output
// exactly with no external process.
package hashing

import (
	"crypto/sha256"
	"encoding/hex"
)

// SHA256Hex returns the lowercase-hex SHA-256 digest of data, matching
// `shasum -a 256` / `sha256sum` output (the value before the filename).
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SHA256HexString is SHA256Hex over a string's UTF-8 bytes (parity with
// context_sha256_string, which hashes a shell string via `printf '%s'`).
func SHA256HexString(s string) string {
	return SHA256Hex([]byte(s))
}
