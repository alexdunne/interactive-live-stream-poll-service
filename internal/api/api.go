package api

import (
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

func InternalServerErrorResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}
}
