package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Issuer       = "oficina-lambda-auth"
	RoleCustomer = "customer"
)

var ErrInvalidToken = errors.New("invalid token")

type Subject struct {
	ID   string
	CPF  string
	Name string
}

type Claims struct {
	Subject string
	CPF     string
	Name    string
	Role    string
}

type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewIssuer(secret string, ttl time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), ttl: ttl, now: time.Now}
}

func (i *TokenIssuer) TTL() time.Duration {
	return i.ttl
}

func (i *TokenIssuer) Issue(subject Subject) (string, error) {
	issuedAt := i.now()
	claims := jwt.MapClaims{
		"sub":  subject.ID,
		"cpf":  subject.CPF,
		"name": subject.Name,
		"role": RoleCustomer,
		"iss":  Issuer,
		"iat":  issuedAt.Unix(),
		"exp":  issuedAt.Add(i.ttl).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
}

type TokenVerifier struct {
	secret []byte
}

func NewVerifier(secret string) *TokenVerifier {
	return &TokenVerifier{secret: []byte(secret)}
}

func (v *TokenVerifier) Verify(raw string) (Claims, error) {
	parsed, err := jwt.Parse(raw, v.keyFunc, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	if !parsed.Valid {
		return Claims{}, ErrInvalidToken
	}

	mapClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return Claims{}, ErrInvalidToken
	}

	return Claims{
		Subject: stringClaim(mapClaims, "sub"),
		CPF:     stringClaim(mapClaims, "cpf"),
		Name:    stringClaim(mapClaims, "name"),
		Role:    stringClaim(mapClaims, "role"),
	}, nil
}

func (v *TokenVerifier) keyFunc(_ *jwt.Token) (interface{}, error) {
	return v.secret, nil
}

func stringClaim(claims jwt.MapClaims, key string) string {
	value, ok := claims[key].(string)
	if !ok {
		return ""
	}
	return value
}
