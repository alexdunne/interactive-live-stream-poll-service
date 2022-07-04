package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/alexdunne/interactive-live-stream-poll-service/internal/api"
	"github.com/alexdunne/interactive-live-stream-poll-service/internal/validator"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

const _tableNameEnv = "POLL_TABLE_NAME"

type submitPollRequest struct {
	PollID string `json:"pollId" validate:"required"`
	// Hacky way to provide a user id whilst we don't have auth
	UserID string `json:"userId" validate:"required"`
	Answer string `json:"answer" validate:"required"`
}

type vote struct {
	PK       string `json:"PK"`
	SK       string `json:"SK"`
	ID       string `json:"id"`
	ItemType string `json:"itemType"`
	PollID   string `json:"pollId"`
	Answer   string `json:"answer"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	db := dynamodb.New(session.Must(session.NewSession()))

	validate, trans, err := validator.NewValidator("en")
	if err != nil {
		log.Printf("error creating validator: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	var submitPollReq submitPollRequest
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

	voteSubmission := createVoteFromRequest(submitPollReq)

	if err := storeVote(db, tableName, voteSubmission); err != nil {
		return api.InternalServerErrorResponse(), nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusAccepted,
	}, nil
}

func createVoteFromRequest(req submitPollRequest) vote {
	return vote{
		PK:       fmt.Sprintf("POLL#%s", req.PollID),
		SK:       fmt.Sprintf("USER#%s", req.UserID),
		ID:       fmt.Sprintf("VOTE#%s", uuid.NewString()),
		ItemType: "Vote",
		PollID:   req.PollID,
		Answer:   req.Answer,
	}
}

func storeVote(db *dynamodb.DynamoDB, tableName string, v vote) error {
	av, err := dynamodbattribute.MarshalMap(v)
	if err != nil {
		log.Printf("error marshalling new vote: %s", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	}

	_, err = db.PutItem(input)
	if err != nil {
		log.Printf("error calling PutItem for vote: %s", err)
		return err
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
