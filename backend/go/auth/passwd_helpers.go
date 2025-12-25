/* Place: backend/go/auth/passwd_helpers.go */
package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func ComparePassword(storedHash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(plain))
}
