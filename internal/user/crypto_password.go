package user

import "golang.org/x/crypto/bcrypt"

type PasswordEncrypter interface {
	Encrypt(pwd string) ([]byte, error)
	Compare(encrypted, pwd string) error
}

type bcryptEncrypt struct{}

func (bcryptEncrypt) Encrypt(pwd string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
}

func (bcryptEncrypt) Compare(encrypted, pwd string) error {
	return bcrypt.CompareHashAndPassword([]byte(encrypted), []byte(pwd))
}
