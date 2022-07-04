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

type createPollRequest struct {
	Question   string   `json:"question" validate:"required,min=1,max=100"`
	Options    []string `json:"options" validate:"required,dive,required,min=1,max=100"`
	ChannelARN string   `json:"channelARN" validate:"required"`
}

type createPollResponse struct {
	ID string `json:"id"`
}

type poll struct {
	PK                   string         `json:"PK"`
	SK                   string         `json:"SK"`
	ID                   string         `json:"id"`
	ItemType             string         `json:"itemType"`
	Question             string         `json:"question"`
	Options              []pollOption   `json:"options"`
	ChannelARN           string         `json:"channelARN"`
	AggregatedVoteTotals map[string]int `json:"aggregatedVoteTotals"`
}

type pollOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	var createPollReq createPollRequest
	if err := json.Unmarshal([]byte(request.Body), &createPollReq); err != nil {
		log.Printf("error unmarshalling request body: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	if err = validate.Struct(createPollReq); err != nil {
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

	newPoll := createPollFromPollRequest(createPollReq)

	if err := storeNewPoll(db, tableName, newPoll); err != nil {
		return api.InternalServerErrorResponse(), nil
	}

	res, err := json.Marshal(createPollResponse{ID: newPoll.ID})
	if err != nil {
		log.Printf("error marshalling poll response: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(res),
		StatusCode: http.StatusAccepted,
	}, nil
}

func createPollFromPollRequest(req createPollRequest) poll {
	id := uuid.NewString()
	dynamoID := fmt.Sprintf("POLL#%s", id)

	var pollOptions []pollOption
	for _, o := range req.Options {
		pollOptions = append(pollOptions, pollOption{
			ID:    uuid.NewString(),
			Label: o,
		})
	}

	totals := make(map[string]int, len(pollOptions))
	for _, o := range pollOptions {
		totals[o.ID] = 0
	}

	return poll{
		PK:                   dynamoID,
		SK:                   dynamoID,
		ID:                   id,
		ItemType:             "Poll",
		Question:             req.Question,
		Options:              pollOptions,
		ChannelARN:           req.ChannelARN,
		AggregatedVoteTotals: totals,
	}
}

func storeNewPoll(db *dynamodb.DynamoDB, tableName string, p poll) error {
	av, err := dynamodbattribute.MarshalMap(p)
	if err != nil {
		log.Printf("error marshalling new poll: %s", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	}

	_, err = db.PutItem(input)
	if err != nil {
		log.Printf("error calling PutItem for poll: %s", err)
		return err
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
