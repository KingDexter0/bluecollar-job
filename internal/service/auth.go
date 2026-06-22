package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	HashPassword(password string) (string, error)
	CheckPassword(password, hash string) error
	GenerateEmployerToken(employerID string) (string, error)
	ParseEmployerToken(tokenString string) (string, error)
}

type authService struct {
	jwtSecret []byte
	issuer    string
	ttl       time.Duration
}

func NewAuthService(jwtSecret, issuer string) AuthService {
	return &authService{
		jwtSecret: []byte(jwtSecret),
		issuer:    issuer,
		ttl:       24 * time.Hour,
	}
}

func (s *authService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *authService) CheckPassword(password, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("%w: invalid credentials", ErrInvalidInput)
	}
	return nil
}

type employerClaims struct {
	EmployerID string `json:"employer_id"`
	jwt.RegisteredClaims
}

func (s *authService) GenerateEmployerToken(employerID string) (string, error) {
	now := time.Now().UTC()
	claims := employerClaims{
		EmployerID: employerID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   employerID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *authService) ParseEmployerToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &employerClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	}, jwt.WithIssuer(s.issuer))
	if err != nil {
		return "", fmt.Errorf("%w: invalid token", ErrInvalidInput)
	}

	claims, ok := token.Claims.(*employerClaims)
	if !ok || !token.Valid || claims.EmployerID == "" {
		return "", fmt.Errorf("%w: invalid token", ErrInvalidInput)
	}
	return claims.EmployerID, nil
}
