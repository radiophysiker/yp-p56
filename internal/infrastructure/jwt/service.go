package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/radiophysiker/d56/internal/domain/user"
)

const (
	TokenExpiration = 24 * time.Hour
)

// Claims represents the JWT claims
// It includes the UserID and standard JWT claims
// This struct is used to create and parse JWT tokens
type Claims struct {
	UserID user.UserID `json:"user_id"`
	jwt.RegisteredClaims
}

// Service provides methods for generating and parsing JWT tokens
type Service struct {
	signingKey []byte
}

// NewService creates a new JWT service with the given secret key
// It returns an error if the secret key is empty
func NewService(jwtSecretKey string) (*Service, error) {
	if jwtSecretKey == "" {
		return nil, fmt.Errorf("JWT secret key cannot be empty")
	}

	return &Service{
		signingKey: []byte(jwtSecretKey),
	}, nil
}

// GenerateToken creates a new JWT token for the given user ID
func (s *Service) GenerateToken(userID user.UserID) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.signingKey)
}

// ParseToken parses the JWT token string and returns the claims
// It returns an error if the token is invalid or expired
func (s *Service) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.signingKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
