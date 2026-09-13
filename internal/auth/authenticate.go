package auth

import (
	"context"
	"errors"

	"github.com/gabrielcamargo/oficina-lambda-auth/internal/cpf"
	"github.com/gabrielcamargo/oficina-lambda-auth/internal/customer"
	"github.com/gabrielcamargo/oficina-lambda-auth/internal/token"
)

var ErrCustomerInactive = errors.New("customer is not active")

type Result struct {
	Token        string
	ExpiresIn    int
	CustomerID   string
	CustomerName string
}

type Authenticator struct {
	customers customer.Repository
	issuer    *token.TokenIssuer
}

func NewAuthenticator(customers customer.Repository, issuer *token.TokenIssuer) *Authenticator {
	return &Authenticator{customers: customers, issuer: issuer}
}

func (a *Authenticator) Authenticate(ctx context.Context, rawCPF string) (Result, error) {
	digits, err := cpf.Validate(rawCPF)
	if err != nil {
		return Result{}, err
	}

	found, err := a.customers.FindByDocument(ctx, digits)
	if err != nil {
		return Result{}, err
	}

	if !found.IsActive() {
		return Result{}, ErrCustomerInactive
	}

	signed, err := a.issuer.Issue(token.Subject{ID: found.ID, CPF: digits, Name: found.Name})
	if err != nil {
		return Result{}, err
	}

	return Result{
		Token:        signed,
		ExpiresIn:    int(a.issuer.TTL().Seconds()),
		CustomerID:   found.ID,
		CustomerName: found.Name,
	}, nil
}
