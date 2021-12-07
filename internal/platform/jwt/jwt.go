package jwt

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"strings"
	"time"
)

var (
	ErrGenerateToken = errors.New("cannot generate token")
	ErrInvalidToken  = errors.New("the received token is invalid")
	ErrTokenExpired  = errors.New("the received token is expired")
	ErrInvalidClaims = errors.New("cannot parse claims")
)

const (
	expKey    = "exp"
	iatKey    = "iat"
	userIDKey = "user_id"
	roleKey   = "role"
)

// GenerateToken will return a jwt generated token with an expiration date, to the user id and with the role received
func GenerateToken(userid int64, role string) (string, error) {
	claims := jwt.MapClaims{
		expKey:    time.Now().Add(time.Minute * 20).Unix(),
		iatKey:    time.Now().Unix(),
		userIDKey: userid,
		roleKey:   role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString([]byte("jdnfksdmfksd"))
	if err != nil {
		return "", fmt.Errorf("%w : %s", ErrGenerateToken, err.Error())
	}

	return t, nil
}

//ValidateToken validate the received token
func ValidateToken(token string) (*jwt.Token, error) {
	//2nd arg function return secret key after checking if the signing method is HMAC and returned key is used by 'Parse' to decode the token)
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			//nil secret key
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte("jdnfksdmfksd"), nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w : %s", ErrInvalidToken, err.Error())
	}

	return parsedToken, nil
}

type Claims struct {
	Iat        int64
	Expiration int64
	UserID     int64
	Role       string
}

// GetClaims return claims from token
func GetClaims(token *jwt.Token) (Claims, error) {
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return Claims{
			Iat:        int64(claims[iatKey].(float64)),
			Expiration: int64(claims[expKey].(float64)),
			UserID:     int64(claims[userIDKey].(float64)),
			Role:       claims[roleKey].(string),
		}, nil
	}

	return Claims{}, ErrInvalidClaims
}
