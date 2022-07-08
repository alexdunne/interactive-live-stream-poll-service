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

type submitVoteRequest struct {
	PollID string `json:"pollId" validate:"required"`
	// Hacky way to provide a user id whilst we don't have auth
	UserID string `json:"userId" validate:"required"`
	Answer string `json:"answer" validate:"required"`
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	pollID, ok := request.PathParameters["id"]
	if !ok {
		return api.ClientError(http.StatusBadRequest, "path parameter 'id' required")
	}

	tableName, ok := os.LookupEnv(_tableNameEnv)
	if !ok {
		return api.ServerError(fmt.Errorf("error environment variable %s not set", _tableNameEnv))
	}

	repo := repository.New(tableName, &db)
	broadcaster := broadcast.New(&ivsClient)

	svc := service.New(repo, broadcaster)

	validate, trans, err := validator.NewValidator("en")
	if err != nil {
		return api.ServerError(fmt.Errorf("error creating validator: %s", err))
	}

	var submitPollReq submitVoteRequest
	if err := json.Unmarshal([]byte(request.Body), &submitPollReq); err != nil {
		return api.ServerError(fmt.Errorf("error unmarshalling request body: %s", err))
	}
	submitPollReq.PollID = pollID

	if err = validate.Struct(submitPollReq); err != nil {
		errMap := validator.ExtractErrorMap(trans, err)

		jsonErrMap, err := json.Marshal(errMap)
		if err != nil {
			return api.ServerError(fmt.Errorf("error: %w", err))
		}

		return api.ClientError(http.StatusBadRequest, string(jsonErrMap))
	}

	_, err = svc.CreatePollVote(ctx, service.NewPollVote{
		PollID: submitPollReq.PollID,
		UserID: submitPollReq.UserID,
		Answer: submitPollReq.Answer,
	})
	if err != nil {
		return api.ServerError(fmt.Errorf("error creating poll bote: %s", err))
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusAccepted,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
