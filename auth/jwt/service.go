package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Service struct {
	secret string
	ttl    int
}

func NewService(secret string, ttl int) *Service {
	return &Service{
		secret: secret,
		ttl:    ttl,
	}
}

func (j *Service) GenerateToken(userID string) (string, error) {
	now := time.Now()

	// jti keeps every token distinct. exp/iat are whole seconds (RFC 7519
	// NumericDate), so without it two tokens minted for the same user in the
	// same second carry identical claims and sign to a byte-identical string.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"jti":     uuid.NewString(),
		"exp":     now.Add(time.Duration(j.ttl) * time.Second).Unix(),
		"iat":     now.Unix(),
	})

	return token.SignedString([]byte(j.secret))
}

func (j *Service) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secret), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("user_id not found in token")
	}

	return userID, nil
}

// TTL returns the access token lifetime in seconds.
func (j *Service) TTL() int {
	return j.ttl
}
