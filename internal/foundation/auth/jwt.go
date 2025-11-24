package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims holds JWT token claims
type TokenClaims struct {
	Subject    string
	Issuer     string
	Audience   string
	ExpMinutes int
}

// GenerateToken generates a JWT token with the given claims and secret
func GenerateToken(claims TokenClaims, secret string) (string, error) {
	now := time.Now()

	jwtClaims := jwt.MapClaims{
		"sub": claims.Subject,
		"iss": claims.Issuer,
		"aud": claims.Audience,
		"iat": now.Unix(),
		"exp": now.Add(time.Duration(claims.ExpMinutes) * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}
