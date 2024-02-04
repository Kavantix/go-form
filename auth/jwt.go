package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired = errors.New("token is expired")
)

var jwtParser = jwt.NewParser(
	jwt.WithValidMethods([]string{
		jwt.SigningMethodEdDSA.Alg(),
	}),
	jwt.WithIssuedAt(),
	jwt.WithExpirationRequired(),
	jwt.WithLeeway(time.Second*10),
	jwt.WithIssuer("go-form"),
)

func keyFunc(t *jwt.Token) (any, error) {
	return publicKey, nil
}

func ParseJwt(tokenString string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	_, err := jwtParser.ParseWithClaims(tokenString, claims, keyFunc)
	if errors.Is(err, jwt.ErrTokenExpired) {
		return claims, ErrTokenExpired
	}
	if err != nil {
		return nil, err
	}
	return claims, nil
}

type JwtOptions struct {
	Audience    string
	Subject     string
	ValidFor    time.Duration
	ExtraClaims map[string]string
}

func (o *JwtOptions) claims() jwt.MapClaims {
	claims := jwt.MapClaims{
		"iss": "go-form",
		"iat": time.Now().Unix(),
	}

	if o.Audience != "" {
		claims["aud"] = o.Audience
	}
	if o.Subject != "" {
		claims["sub"] = o.Subject
	}
	if o.ExtraClaims != nil {
		claims["extra"] = o.ExtraClaims
	}
	if o.ValidFor > 0 {
		claims["exp"] = time.Now().Add(o.ValidFor).Unix()
	} else {
		claims["exp"] = time.Now().Add(time.Minute * 5).Unix()
	}

	return claims
}

func CreateJwt(options *JwtOptions) (string, error) {
	var claims jwt.MapClaims
	if options != nil {
		claims = options.claims()
	} else {
		claims = jwt.MapClaims{}
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	result, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign jwt: %w", err)
	}
	return result, nil
}
