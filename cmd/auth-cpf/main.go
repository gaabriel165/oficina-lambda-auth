package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gabrielcamargo/oficina-lambda-auth/internal/auth"
	"github.com/gabrielcamargo/oficina-lambda-auth/internal/cpf"
	"github.com/gabrielcamargo/oficina-lambda-auth/internal/customer"
	"github.com/gabrielcamargo/oficina-lambda-auth/internal/httpresponse"
	"github.com/gabrielcamargo/oficina-lambda-auth/internal/token"
)

const (
	defaultTokenTTLSeconds = 3600
	customerLookupTimeout  = 5 * time.Second
)

type authRequest struct {
	CPF string `json:"cpf"`
}

type customerResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type authResponse struct {
	Token     string           `json:"token"`
	TokenType string           `json:"token_type"`
	ExpiresIn int              `json:"expires_in"`
	Customer  customerResponse `json:"customer"`
}

var (
	logger        *slog.Logger
	authenticator *auth.Authenticator
)

func main() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("function", "auth-cpf")

	customers := customer.NewHTTPRepository(
		&http.Client{Timeout: customerLookupTimeout},
		os.Getenv("CUSTOMER_API_URL"),
		os.Getenv("INTERNAL_API_SECRET"),
	)
	issuer := token.NewIssuer(os.Getenv("JWT_SECRET"), tokenTTL())
	authenticator = auth.NewAuthenticator(customers, issuer)

	lambda.Start(handle)
}

func handle(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	requestLogger := logger.With("request_id", request.RequestContext.RequestID)

	var body authRequest
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		requestLogger.Warn("authentication rejected", "event", "auth.cpf", "outcome", "invalid_body")
		return httpresponse.Error(http.StatusBadRequest, "invalid request body"), nil
	}

	digits, cpfErr := cpf.Validate(body.CPF)
	maskedCPF := cpf.Mask(digits)
	if cpfErr != nil {
		requestLogger.Warn("authentication rejected", "event", "auth.cpf", "cpf_masked", maskedCPF, "outcome", "invalid_cpf")
		return httpresponse.Error(http.StatusBadRequest, "invalid cpf"), nil
	}

	result, err := authenticator.Authenticate(ctx, digits)
	if err != nil {
		return mapError(requestLogger, maskedCPF, err), nil
	}

	requestLogger.Info("authentication succeeded", "event", "auth.cpf", "cpf_masked", maskedCPF, "outcome", "success", "customer_id", result.CustomerID)
	return httpresponse.JSON(http.StatusOK, authResponse{
		Token:     result.Token,
		TokenType: "Bearer",
		ExpiresIn: result.ExpiresIn,
		Customer:  customerResponse{ID: result.CustomerID, Name: result.CustomerName},
	}), nil
}

func mapError(requestLogger *slog.Logger, maskedCPF string, err error) events.APIGatewayV2HTTPResponse {
	if errors.Is(err, customer.ErrCustomerNotFound) {
		requestLogger.Warn("authentication rejected", "event", "auth.cpf", "cpf_masked", maskedCPF, "outcome", "customer_not_found")
		return httpresponse.Error(http.StatusNotFound, "customer not found")
	}
	if errors.Is(err, auth.ErrCustomerInactive) {
		requestLogger.Warn("authentication rejected", "event", "auth.cpf", "cpf_masked", maskedCPF, "outcome", "customer_inactive")
		return httpresponse.Error(http.StatusForbidden, "customer is not active")
	}
	if errors.Is(err, customer.ErrCustomerServiceUnavailable) {
		requestLogger.Error("authentication failed", "event", "auth.cpf", "cpf_masked", maskedCPF, "outcome", "customer_service_unavailable", "error", err.Error())
		return httpresponse.Error(http.StatusBadGateway, "customer service unavailable")
	}
	requestLogger.Error("authentication failed", "event", "auth.cpf", "cpf_masked", maskedCPF, "outcome", "error", "error", err.Error())
	return httpresponse.Error(http.StatusInternalServerError, "internal server error")
}

func tokenTTL() time.Duration {
	seconds, err := strconv.Atoi(os.Getenv("TOKEN_TTL_SECONDS"))
	if err != nil || seconds <= 0 {
		return defaultTokenTTLSeconds * time.Second
	}
	return time.Duration(seconds) * time.Second
}
