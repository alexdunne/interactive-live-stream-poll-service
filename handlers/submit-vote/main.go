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

type submitVoteRequest struct {
	PollID string `json:"pollId" validate:"required"`
	// Hacky way to provide a user id whilst we don't have auth
	UserID string `json:"userId" validate:"required"`
	Answer string `json:"answer" validate:"required"`
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	pollID, ok := request.PathParameters["id"]
	if !ok {
		return events.APIGatewayProxyResponse{
			Body:       "path parameter 'id' required",
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	tableName, ok := os.LookupEnv(_tableNameEnv)
	if !ok {
		log.Printf("error environment variable %s not set", _tableNameEnv)
		return api.InternalServerErrorResponse(), nil
	}

	sess := session.Must(session.NewSession())
	db := dynamodb.New(sess)
	ivsSvc := ivs.New(sess)

	repo := repository.New(tableName, db)
	broadcaster := broadcast.New(ivsSvc)

	svc := service.New(repo, broadcaster)

	validate, trans, err := validator.NewValidator("en")
	if err != nil {
		log.Printf("error creating validator: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	var submitPollReq submitVoteRequest
	if err := json.Unmarshal([]byte(request.Body), &submitPollReq); err != nil {
		log.Printf("error unmarshalling request body: %s", err)
		return api.InternalServerErrorResponse(), nil
	}
	submitPollReq.PollID = pollID

	if err = validate.Struct(submitPollReq); err != nil {
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

	_, err = svc.CreatePollVote(ctx, service.NewPollVote{
		PollID: submitPollReq.PollID,
		UserID: submitPollReq.UserID,
		Answer: submitPollReq.Answer,
	})
	if err != nil {
		log.Printf("error creating poll vote: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusAccepted,
	}, nil
}

func main() {
	lambda.Start(handler)
}
