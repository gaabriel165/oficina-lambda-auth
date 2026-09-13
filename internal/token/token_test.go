package token_test

import (
	"testing"
	"time"

	"github.com/gabrielcamargo/oficina-lambda-auth/internal/token"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secret = "test-secret"

func TestIssue_ShouldProduceVerifiableTokenWithCustomerClaims(t *testing.T) {
	issuer := token.NewIssuer(secret, time.Hour)

	raw, err := issuer.Issue(token.Subject{ID: "customer-1", CPF: "52998224725", Name: "John Doe"})
	require.NoError(t, err)

	claims, err := token.NewVerifier(secret).Verify(raw)

	require.NoError(t, err)
	assert.Equal(t, "customer-1", claims.Subject)
	assert.Equal(t, "52998224725", claims.CPF)
	assert.Equal(t, "John Doe", claims.Name)
	assert.Equal(t, token.RoleCustomer, claims.Role)
}

func TestVerify_ShouldRejectTokenSignedWithAnotherSecret(t *testing.T) {
	raw, err := token.NewIssuer("other-secret", time.Hour).Issue(token.Subject{ID: "customer-1"})
	require.NoError(t, err)

	_, err = token.NewVerifier(secret).Verify(raw)

	assert.ErrorIs(t, err, token.ErrInvalidToken)
}

func TestVerify_ShouldRejectExpiredToken(t *testing.T) {
	raw, err := token.NewIssuer(secret, -time.Minute).Issue(token.Subject{ID: "customer-1"})
	require.NoError(t, err)

	_, err = token.NewVerifier(secret).Verify(raw)

	assert.ErrorIs(t, err, token.ErrInvalidToken)
}

func TestVerify_ShouldRejectTokenWithoutExpiration(t *testing.T) {
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "customer-1"}).SignedString([]byte(secret))
	require.NoError(t, err)

	_, err = token.NewVerifier(secret).Verify(raw)

	assert.ErrorIs(t, err, token.ErrInvalidToken)
}

func TestVerify_ShouldRejectMalformedToken(t *testing.T) {
	_, err := token.NewVerifier(secret).Verify("not-a-jwt")

	assert.ErrorIs(t, err, token.ErrInvalidToken)
}
