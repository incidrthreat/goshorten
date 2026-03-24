package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims for a GoShorten user.
type Claims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT creation and validation.
type JWTManager struct {
	Secret   []byte
	ExpiryHr int
}

// NewJWTManager creates a new JWT manager.
func NewJWTManager(secret string, expiryHours int) *JWTManager {
	return &JWTManager{
		Secret:   []byte(secret),
		ExpiryHr: expiryHours,
	}
}

// Generate creates a signed JWT for the given user.
// Returns (tokenString, jti, error). The jti uniquely identifies this session.
func (m *JWTManager) Generate(userID int64, email, role string) (string, string, error) {
	jti, err := generateJTI()
	if err != nil {
		return "", "", fmt.Errorf("generate jti: %w", err)
	}

	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(m.ExpiryHr) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "goshorten",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.Secret)
	if err != nil {
		return "", "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, jti, nil
}

func generateJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Verify parses and validates a JWT, returning the claims.
func (m *JWTManager) Verify(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.Secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
