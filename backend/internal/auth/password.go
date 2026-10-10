package auth

import "golang.org/x/crypto/bcrypt"

// BcryptCost is the explicit work factor used for every employee password. It is
// deliberately stated here (rather than relying on bcrypt.DefaultCost) so the
// cost is a documented, reviewable constant (AC-24).
const BcryptCost = 12

// HashPassword returns the bcrypt hash of password. The comparison cost is
// BcryptCost and bcrypt salts every hash, so hashing the same password twice
// yields two different hashes.
//
// The plaintext is used only here and is never returned, logged or stored.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword reports whether password matches the stored hash. A malformed
// or unknown hash verifies as false, never as an error the caller might leak.
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
