package customer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielcamargo/oficina-lambda-auth/internal/customer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSecret   = "internal-secret"
	testDocument = "52998224725"
)

func newCustomerService(t *testing.T, handler http.HandlerFunc) *customer.HTTPRepository {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return customer.NewHTTPRepository(server.Client(), server.URL+"/", testSecret)
}

func TestFindByDocument_ShouldReturnCustomerFromTheOrderService(t *testing.T) {
	var receivedPath, receivedSecret string
	repository := newCustomerService(t, func(writer http.ResponseWriter, request *http.Request) {
		receivedPath = request.URL.Path
		receivedSecret = request.Header.Get("X-Internal-Secret")
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"customer-1","name":"Maria Silva","status":"active"}`))
	})

	found, err := repository.FindByDocument(context.Background(), testDocument)

	require.NoError(t, err)
	assert.Equal(t, "customer-1", found.ID)
	assert.Equal(t, "Maria Silva", found.Name)
	assert.True(t, found.IsActive())
	assert.Equal(t, "/internal/customers/by-document/"+testDocument, receivedPath)
	assert.Equal(t, testSecret, receivedSecret)
}

func TestFindByDocument_ShouldReturnNotFoundWhenCustomerDoesNotExist(t *testing.T) {
	repository := newCustomerService(t, func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
	})

	_, err := repository.FindByDocument(context.Background(), testDocument)

	assert.ErrorIs(t, err, customer.ErrCustomerNotFound)
}

func TestFindByDocument_ShouldReturnUnavailableWhenSecretIsRejected(t *testing.T) {
	repository := newCustomerService(t, func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	})

	_, err := repository.FindByDocument(context.Background(), testDocument)

	assert.ErrorIs(t, err, customer.ErrCustomerServiceUnavailable)
}

func TestFindByDocument_ShouldReturnUnavailableWhenBodyIsMalformed(t *testing.T) {
	repository := newCustomerService(t, func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("not-json"))
	})

	_, err := repository.FindByDocument(context.Background(), testDocument)

	assert.ErrorIs(t, err, customer.ErrCustomerServiceUnavailable)
}

func TestFindByDocument_ShouldReturnUnavailableWhenServiceIsDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := customer.NewHTTPRepository(server.Client(), server.URL, testSecret)
	server.Close()

	_, err := repository.FindByDocument(context.Background(), testDocument)

	assert.ErrorIs(t, err, customer.ErrCustomerServiceUnavailable)
}
