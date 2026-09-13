package customer

import (
	"context"
	"errors"
)

const StatusActive = "active"

var ErrCustomerNotFound = errors.New("customer not found")

type Customer struct {
	ID     string
	Name   string
	Status string
}

func (c Customer) IsActive() bool {
	return c.Status == StatusActive
}

type Repository interface {
	FindByDocument(ctx context.Context, document string) (*Customer, error)
}
