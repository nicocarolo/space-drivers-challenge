package user

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_bcryptpassword(t *testing.T) {
	bcrypt := bcryptEncrypt{}
	pwd, err := bcrypt.Encrypt("mock password")

	assert.Nil(t, err)
	assert.NotNil(t, pwd)

	// test compare same password, successful
	err = bcrypt.Compare(string(pwd), "mock password")
	assert.Nil(t, err)

	// test compare wrong password, failed
	err = bcrypt.Compare(string(pwd), "wrong password")
	assert.NotNil(t, err)
}
