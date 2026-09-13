package httpresponse

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

var jsonHeaders = map[string]string{"Content-Type": "application/json"}

func JSON(status int, body any) events.APIGatewayV2HTTPResponse {
	encoded, err := json.Marshal(body)
	if err != nil {
		return Error(http.StatusInternalServerError, "internal server error")
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers:    jsonHeaders,
		Body:       string(encoded),
	}
}

func Error(status int, message string) events.APIGatewayV2HTTPResponse {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers:    jsonHeaders,
		Body:       fmt.Sprintf(`{"error":%q}`, message),
	}
}
