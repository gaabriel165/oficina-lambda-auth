package customer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	internalSecretHeader = "X-Internal-Secret"
	lookupPath           = "/internal/customers/by-document/"
)

var ErrCustomerServiceUnavailable = errors.New("customer service unavailable")

type customerPayload struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type HTTPRepository struct {
	client  *http.Client
	baseURL string
	secret  string
}

func NewHTTPRepository(client *http.Client, baseURL, secret string) *HTTPRepository {
	return &HTTPRepository{client: client, baseURL: strings.TrimRight(baseURL, "/"), secret: secret}
}

func (r *HTTPRepository) FindByDocument(ctx context.Context, document string) (*Customer, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+lookupPath+url.PathEscape(document), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set(internalSecretHeader, r.secret)

	response, err := r.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCustomerServiceUnavailable, err.Error())
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, ErrCustomerNotFound
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrCustomerServiceUnavailable, response.StatusCode)
	}

	var payload customerPayload
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCustomerServiceUnavailable, err.Error())
	}
	return &Customer{ID: payload.ID, Name: payload.Name, Status: payload.Status}, nil
}
