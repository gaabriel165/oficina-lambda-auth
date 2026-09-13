package customer

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const findByDocumentQuery = "SELECT id, name, status FROM customers WHERE document = $1"

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) FindByDocument(ctx context.Context, document string) (*Customer, error) {
	var found Customer
	err := r.pool.QueryRow(ctx, findByDocumentQuery, document).Scan(&found.ID, &found.Name, &found.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCustomerNotFound
	}
	if err != nil {
		return nil, err
	}
	return &found, nil
}
