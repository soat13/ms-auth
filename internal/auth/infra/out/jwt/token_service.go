package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/soat13/oficina-auth/internal/auth/application"
	authDomain "github.com/soat13/oficina-auth/internal/auth/domain"
)

const (
	defaultIssuer = "oficina-api"
)

var (
	ErrMissingSecret = errors.New("jwt secret must be provided")
)

type authClaims struct {
	UserID string   `json:"uid"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwtlib.RegisteredClaims
}

type Service struct {
	secret     []byte
	expiration time.Duration
	issuer     string
	now        func() time.Time
}

func NewTokenService(secret string, expiration time.Duration) (application.TokenService, error) {
	if secret == "" {
		return nil, ErrMissingSecret
	}

	if expiration <= 0 {
		expiration = time.Hour
	}

	return &Service{
		secret:     []byte(secret),
		expiration: expiration,
		issuer:     defaultIssuer,
		now:        time.Now,
	}, nil
}

func (s *Service) Generate(_ context.Context, user *application.UserView) (authDomain.Token, error) {
	now := s.now()
	expiresAt := now.Add(s.expiration)

	claims := authClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Roles:  user.Roles.Strings(),
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    s.issuer,
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(expiresAt),
			Subject:   user.ID.String(),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return authDomain.Token{}, fmt.Errorf("sign token: %w", err)
	}

	return authDomain.Token{
		Value:     signed,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) Validate(_ context.Context, token string) (authDomain.Claims, error) {
	var claims authClaims

	parsed, err := jwtlib.ParseWithClaims(
		token,
		&claims,
		func(_ *jwtlib.Token) (interface{}, error) {
			return s.secret, nil
		},
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return authDomain.Claims{}, authDomain.ErrExpiredToken
		}
		return authDomain.Claims{}, authDomain.ErrInvalidToken
	}
	if parsed == nil || !parsed.Valid {
		return authDomain.Claims{}, authDomain.ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return authDomain.Claims{}, authDomain.ErrInvalidToken
	}

	return authDomain.Claims{
		UserID: userID,
		Email:  claims.Email,
		Roles:  claims.Roles,
		Exp:    claims.ExpiresAt.Time,
	}, nil
}
