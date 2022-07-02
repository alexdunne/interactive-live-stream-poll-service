package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/alexdunne/interactive-live-stream-poll-service/internal/validator"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type createPollRequest struct {
	Question   string   `json:"question" validate:"required,min=2,max=100"`
	Options    []string `json:"options" validate:"required,dive,required,min=2,max=100"`
	ChannelARN string   `json:"channelARN" validate:"required,min=20,max=2048"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	validate, err := validator.NewValidator("en")
	if err != nil {
		log.Printf("creating validator: %s", err)

		return events.APIGatewayProxyResponse{
			Body:       http.StatusText(http.StatusInternalServerError),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	var createPollReq createPollRequest

	if err := json.Unmarshal([]byte(request.Body), &createPollReq); err != nil {
		log.Printf("unmarshalling request body: %s", err)

		return events.APIGatewayProxyResponse{
			Body:       http.StatusText(http.StatusInternalServerError),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	if err = validate.Struct(createPollReq); err != nil {
		return events.APIGatewayProxyResponse{
			Body:       http.StatusText(http.StatusInternalServerError),
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       "hi",
		StatusCode: http.StatusAccepted,
	}, nil
}

func main() {
	lambda.Start(handler)
}
