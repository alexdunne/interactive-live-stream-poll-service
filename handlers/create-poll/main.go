package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/alexdunne/interactive-live-stream-poll-service/internal/api"
	"github.com/alexdunne/interactive-live-stream-poll-service/internal/broadcast"
	"github.com/alexdunne/interactive-live-stream-poll-service/internal/repository"
	"github.com/alexdunne/interactive-live-stream-poll-service/internal/service"
	"github.com/alexdunne/interactive-live-stream-poll-service/internal/validator"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ivs"
)

const _tableNameEnv = "POLL_TABLE_NAME"

type createPollRequest struct {
	Question   string   `json:"question" validate:"required,min=1,max=100"`
	Options    []string `json:"options" validate:"required,dive,required,min=1,max=100"`
	ChannelARN string   `json:"channelARN" validate:"required"`
}

type createPollResponse struct {
	ID string `json:"id"`
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	tableName, ok := os.LookupEnv(_tableNameEnv)
	if !ok {
		log.Printf("error environment variable %s not set", _tableNameEnv)
		return api.InternalServerErrorResponse(), nil
	}

	validate, trans, err := validator.NewValidator("en")
	if err != nil {
		log.Printf("error creating validator: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	sess := session.Must(session.NewSession())
	db := dynamodb.New(sess)
	ivsSvc := ivs.New(sess)

	repo := repository.New(tableName, db)
	broadcaster := broadcast.New(ivsSvc)

	svc := service.New(repo, broadcaster)

	var createPollReq createPollRequest
	if err := json.Unmarshal([]byte(request.Body), &createPollReq); err != nil {
		log.Printf("error unmarshalling request body: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	if err := validate.Struct(createPollReq); err != nil {
		errMap := validator.ExtractErrorMap(trans, err)

		jsonErrMap, err := json.Marshal(errMap)
		if err != nil {
			return api.InternalServerErrorResponse(), nil
		}

		return events.APIGatewayProxyResponse{
			Body:       string(jsonErrMap),
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	poll, err := svc.CreatePoll(ctx, service.NewPoll{
		Question:   createPollReq.Question,
		Options:    createPollReq.Options,
		ChannelARN: createPollReq.ChannelARN,
	})
	if err != nil {
		log.Printf("error creating poll: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	res, err := json.Marshal(createPollResponse{ID: poll.ID})
	if err != nil {
		log.Printf("error marshalling poll response: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(res),
		StatusCode: http.StatusAccepted,
	}, nil
}

func main() {
	lambda.Start(handler)
}
