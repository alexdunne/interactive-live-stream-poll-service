package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ivs"
)

const _tableNameEnv = "POLL_TABLE_NAME"

var db dynamodb.Client
var ivsClient ivs.Client

func init() {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	db = *dynamodb.NewFromConfig(sdkConfig)
	ivsClient = *ivs.NewFromConfig(sdkConfig)
}

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
		return api.ServerError(fmt.Errorf("error environment variable %s not set", _tableNameEnv))
	}

	validate, trans, err := validator.NewValidator("en")
	if err != nil {
		return api.ServerError(fmt.Errorf("error creating validator: %s", err))
	}

	repo := repository.New(tableName, &db)
	broadcaster := broadcast.New(&ivsClient)

	svc := service.New(repo, broadcaster)

	var createPollReq createPollRequest
	if err := json.Unmarshal([]byte(request.Body), &createPollReq); err != nil {
		return api.ServerError(fmt.Errorf("error unmarshalling request body: %s", err))
	}

	if err := validate.Struct(createPollReq); err != nil {
		errMap := validator.ExtractErrorMap(trans, err)

		jsonErrMap, err := json.Marshal(errMap)
		if err != nil {
			return api.ServerError(fmt.Errorf("error: %w", err))
		}

		return api.ClientError(http.StatusBadRequest, string(jsonErrMap))
	}

	poll, err := svc.CreatePoll(ctx, service.NewPoll{
		Question:   createPollReq.Question,
		Options:    createPollReq.Options,
		ChannelARN: createPollReq.ChannelARN,
	})
	if err != nil {
		return api.ServerError(fmt.Errorf("error creating poll: %s", err))
	}

	res, err := json.Marshal(createPollResponse{ID: poll.ID})
	if err != nil {
		return api.ServerError(fmt.Errorf("error marshalling poll response: %s", err))
	}

	return events.APIGatewayProxyResponse{
		Body:       string(res),
		StatusCode: http.StatusAccepted,
	}, nil
}

func main() {
	lambda.Start(handler)
}
