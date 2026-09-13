package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabrielcamargo/oficina-lambda-auth/internal/auth"
	"github.com/gabrielcamargo/oficina-lambda-auth/internal/cpf"
	"github.com/gabrielcamargo/oficina-lambda-auth/internal/customer"
	"github.com/gabrielcamargo/oficina-lambda-auth/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validCPF = "52998224725"

type fakeRepository struct {
	customers map[string]*customer.Customer
	err       error
}

func (f *fakeRepository) FindByDocument(_ context.Context, document string) (*customer.Customer, error) {
	if f.err != nil {
		return nil, f.err
	}
	found, ok := f.customers[document]
	if !ok {
		return nil, customer.ErrCustomerNotFound
	}
	return found, nil
}

func newAuthenticator(repository customer.Repository) *auth.Authenticator {
	return auth.NewAuthenticator(repository, token.NewIssuer("secret", time.Hour))
}

func TestAuthenticate_ShouldIssueTokenForActiveCustomer(t *testing.T) {
	repository := &fakeRepository{customers: map[string]*customer.Customer{
		validCPF: {ID: "customer-1", Name: "John Doe", Status: customer.StatusActive},
	}}

	result, err := newAuthenticator(repository).Authenticate(context.Background(), "529.982.247-25")

	require.NoError(t, err)
	assert.NotEmpty(t, result.Token)
	assert.Equal(t, 3600, result.ExpiresIn)
	assert.Equal(t, "customer-1", result.CustomerID)
	assert.Equal(t, "John Doe", result.CustomerName)
}

func TestAuthenticate_ShouldReturnErrorWhenCPFIsInvalid(t *testing.T) {
	repository := &fakeRepository{customers: map[string]*customer.Customer{}}

	_, err := newAuthenticator(repository).Authenticate(context.Background(), "11111111111")

	assert.ErrorIs(t, err, cpf.ErrInvalidCPF)
}

func TestAuthenticate_ShouldReturnErrorWhenCustomerNotFound(t *testing.T) {
	repository := &fakeRepository{customers: map[string]*customer.Customer{}}

	_, err := newAuthenticator(repository).Authenticate(context.Background(), validCPF)

	assert.ErrorIs(t, err, customer.ErrCustomerNotFound)
}

func TestAuthenticate_ShouldReturnErrorWhenCustomerIsInactive(t *testing.T) {
	repository := &fakeRepository{customers: map[string]*customer.Customer{
		validCPF: {ID: "customer-1", Name: "John Doe", Status: "inactive"},
	}}

	_, err := newAuthenticator(repository).Authenticate(context.Background(), validCPF)

	assert.ErrorIs(t, err, auth.ErrCustomerInactive)
}

func TestAuthenticate_ShouldPropagateRepositoryError(t *testing.T) {
	repositoryErr := errors.New("connection refused")
	repository := &fakeRepository{err: repositoryErr}

	_, err := newAuthenticator(repository).Authenticate(context.Background(), validCPF)

	assert.ErrorIs(t, err, repositoryErr)
}
