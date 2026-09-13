package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gabrielcamargo/oficina-lambda-auth/internal/token"
)

const bearerPrefix = "Bearer "

var (
	logger   *slog.Logger
	verifier *token.TokenVerifier
)

func main() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("function", "authorizer")
	verifier = token.NewVerifier(os.Getenv("JWT_SECRET"))
	lambda.Start(handle)
}

func handle(_ context.Context, request events.APIGatewayV2CustomAuthorizerV2Request) (events.APIGatewayV2CustomAuthorizerSimpleResponse, error) {
	requestLogger := logger.With("request_id", request.RequestContext.RequestID, "route", request.RouteKey)

	raw, ok := bearerToken(request)
	if !ok {
		requestLogger.Warn("authorization denied", "event", "auth.authorize", "outcome", "missing_token")
		return deny(), nil
	}

	claims, err := verifier.Verify(raw)
	if err != nil {
		requestLogger.Warn("authorization denied", "event", "auth.authorize", "outcome", "invalid_token")
		return deny(), nil
	}

	requestLogger.Info("authorization granted", "event", "auth.authorize", "outcome", "success", "subject", claims.Subject, "role", claims.Role)
	return events.APIGatewayV2CustomAuthorizerSimpleResponse{
		IsAuthorized: true,
		Context: map[string]interface{}{
			"sub":  claims.Subject,
			"role": claims.Role,
		},
	}, nil
}

func bearerToken(request events.APIGatewayV2CustomAuthorizerV2Request) (string, bool) {
	header := ""
	if len(request.IdentitySource) > 0 {
		header = request.IdentitySource[0]
	}
	if header == "" {
		header = authorizationHeader(request.Headers)
	}
	if !strings.HasPrefix(header, bearerPrefix) {
		return "", false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
	return raw, raw != ""
}

func authorizationHeader(headers map[string]string) string {
	for key, value := range headers {
		if strings.EqualFold(key, "authorization") {
			return value
		}
	}
	return ""
}

func deny() events.APIGatewayV2CustomAuthorizerSimpleResponse {
	return events.APIGatewayV2CustomAuthorizerSimpleResponse{IsAuthorized: false}
}
