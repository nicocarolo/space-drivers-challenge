package user

import "golang.org/x/crypto/bcrypt"

type PasswordEncrypter func(pwd string) ([]byte, error)

// bcryptEncrypter return a PasswordEncrypter who will received a password and return that encrypted with bcrypt
// algorithm with default cost
func bcryptEncrypter() PasswordEncrypter {
	return func(pwd string) ([]byte, error) {
		return bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	}
}
