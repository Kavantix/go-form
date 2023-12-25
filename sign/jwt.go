package sign

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtParser = jwt.NewParser(
	jwt.WithValidMethods([]string{
		jwt.SigningMethodEdDSA.Alg(),
	}),
	jwt.WithIssuedAt(),
	jwt.WithExpirationRequired(),
	jwt.WithLeeway(time.Minute*1),
	jwt.WithIssuer("go-form"),
)

func keyFunc(t *jwt.Token) (any, error) {
	return publicKey, nil
}

func ParseJwt(tokenString string) (*jwt.Token, error) {
	return jwtParser.Parse(tokenString, keyFunc)
}

type JwtOptions struct {
	Subject string
}

func (o *JwtOptions) claims() jwt.MapClaims {
	claims := jwt.MapClaims{
		"iss": "go-form",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * 10).Unix(),
	}

	if o.Subject != "" {
		claims["sub"] = o.Subject
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
