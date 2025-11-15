/* Place: backend/go/auth/passwd_helpers.go */
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword returns bcrypt hashed password using provided cost.
func HashPassword(password string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ComparePassword compares plain password with stored bcrypt hash.
func ComparePassword(storedHash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(plain))
}
