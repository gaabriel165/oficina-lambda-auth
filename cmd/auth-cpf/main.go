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
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultTokenTTLSeconds = 3600
	databaseTimeout        = 5 * time.Second
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

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Error("database pool creation failed", "error", err.Error())
		os.Exit(1)
	}

	issuer := token.NewIssuer(os.Getenv("JWT_SECRET"), tokenTTL())
	authenticator = auth.NewAuthenticator(customer.NewPostgresRepository(pool), issuer)

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

	dbCtx, cancel := context.WithTimeout(ctx, databaseTimeout)
	defer cancel()

	result, err := authenticator.Authenticate(dbCtx, digits)
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
