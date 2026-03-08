package jwt

import (
	"errors"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	TokenType TokenType `json:"token_type"`
	gojwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type JWTService interface {
	GenerateTokenPair(userID uuid.UUID, email, role string) (*TokenPair, error)
	ValidateToken(token string, tokenType TokenType) (*Claims, error)
	ParseUnverified(token string) (*Claims, error)
}

type jwtService struct {
	accessSecret  string
	refreshSecret string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTService(accessSecret, refreshSecret string, accessExpiry, refreshExpiry time.Duration) JWTService {
	return &jwtService{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (s *jwtService) GenerateTokenPair(userID uuid.UUID, email, role string) (*TokenPair, error) {
	now := time.Now()
	accessExpiresAt := now.Add(s.accessExpiry)

	accessClaims := &Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: AccessToken,
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(accessExpiresAt),
		},
	}
	accessTokenStr, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(s.accessSecret))
	if err != nil {
		return nil, err
	}

	refreshClaims := &Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: RefreshToken,
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(s.refreshExpiry)),
		},
	}
	refreshTokenStr, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, refreshClaims).
		SignedString([]byte(s.refreshSecret))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
		ExpiresAt:    accessExpiresAt,
	}, nil
}

func (s *jwtService) ValidateToken(token string, tokenType TokenType) (*Claims, error) {
	secret := s.accessSecret
	if tokenType == RefreshToken {
		secret = s.refreshSecret
	}

	claims := &Claims{}
	parsed, err := gojwt.ParseWithClaims(token, claims, func(t *gojwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != tokenType {
		return nil, errors.New("wrong token type")
	}
	return claims, nil
}

func (s *jwtService) ParseUnverified(token string) (*Claims, error) {
	claims := &Claims{}
	p := gojwt.NewParser()
	_, _, err := p.ParseUnverified(token, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
