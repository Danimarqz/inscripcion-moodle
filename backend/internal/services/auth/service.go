package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/inscripcion-moodle/go-backend/internal/config"
)

type Service struct {
	secretKey string
	algorithm string
	tokenTTL  time.Duration
}

type claims struct {
	Username string `json:"sub"`
	jwt.RegisteredClaims
}

func New(cfg *config.Config) (*Service, error) {
	if cfg.SecretKey == "" {
		return nil, errors.New("secret key is required")
	}
	ttl := time.Duration(cfg.TokenTTLMinutes) * time.Minute
	if ttl <= 0 {
		ttl = time.Minute * 60
	}
	return &Service{
		secretKey: cfg.SecretKey,
		algorithm: cfg.TokenAlgorithm,
		tokenTTL:  ttl,
	}, nil
}

func (s *Service) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *Service) VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (s *Service) CreateToken(username string) (string, error) {
	now := time.Now().UTC()
	claims := claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "inscripcion-moodle",
			Audience:  jwt.ClaimStrings{"inscripcion-moodle-admin"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(s.algorithm), claims)
	return token.SignedString([]byte(s.secretKey))
}

func (s *Service) ParseToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != s.algorithm {
			return nil, errors.New("unexpected signing algorithm")
		}
		return []byte(s.secretKey), nil
	},
		jwt.WithIssuer("inscripcion-moodle"),
		jwt.WithAudience("inscripcion-moodle-admin"),
	)
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		return claims.Username, nil
	}
	return "", errors.New("invalid token")
}
