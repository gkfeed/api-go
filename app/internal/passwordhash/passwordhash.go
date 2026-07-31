// Package passwordhash provides password hashing and verification.
package passwordhash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Version = 19
	memory        = 64 * 1024
	iterations    = 3
	parallelism   = 4
	keyLength     = 32
	saltLength    = 16
)

// HashPassword returns a PHC-formatted Argon2id hash of password.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version,
		memory,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// ComparePassword reports whether password matches an encoded Argon2id hash.
func ComparePassword(encodedHash, password string) bool {
	params, salt, expected, err := decode(encodedHash)
	if err != nil {
		return false
	}

	actual := argon2.IDKey(
		[]byte(password),
		salt,
		params.iterations,
		params.memory,
		params.parallelism,
		uint32(len(expected)),
	)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

// IsEncoded reports whether encodedHash is a valid Argon2id PHC string that
// can be verified by ComparePassword.
func IsEncoded(encodedHash string) bool {
	_, _, _, err := decode(encodedHash)
	return err == nil
}

type parameters struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func decode(encodedHash string) (parameters, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return parameters{}, nil, nil, errors.New("invalid Argon2id hash format")
	}

	params, err := parseParameters(parts[3])
	if err != nil {
		return parameters{}, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return parameters{}, nil, nil, errors.New("invalid Argon2id salt")
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) == 0 {
		return parameters{}, nil, nil, errors.New("invalid Argon2id output")
	}

	return params, salt, hash, nil
}

func parseParameters(encoded string) (parameters, error) {
	var result parameters
	seen := make(map[string]bool, 3)
	for _, field := range strings.Split(encoded, ",") {
		key, value, ok := strings.Cut(field, "=")
		if !ok || seen[key] {
			return parameters{}, errors.New("invalid Argon2id parameters")
		}
		seen[key] = true

		number, err := strconv.ParseUint(value, 10, 32)
		if err != nil || number == 0 {
			return parameters{}, errors.New("invalid Argon2id parameter value")
		}
		switch key {
		case "m":
			result.memory = uint32(number)
		case "t":
			result.iterations = uint32(number)
		case "p":
			if number > 255 {
				return parameters{}, errors.New("invalid Argon2id parallelism")
			}
			result.parallelism = uint8(number)
		default:
			return parameters{}, errors.New("unknown Argon2id parameter")
		}
	}

	if len(seen) != 3 || result.memory < 8*uint32(result.parallelism) {
		return parameters{}, errors.New("invalid Argon2id parameters")
	}
	return result, nil
}
