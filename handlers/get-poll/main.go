package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/alexdunne/interactive-live-stream-poll-service/internal/api"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

const _tableNameEnv = "POLL_TABLE_NAME"

type getPollResponse struct {
	Data poll `json:"data"`
}

type poll struct {
	ID       string       `json:"id"`
	Question string       `json:"question"`
	Options  []pollOption `json:"options"`
}

type pollOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
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

	pollDatabaseKey := fmt.Sprintf("POLL#%s", pollID)

	result, err := db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(pollDatabaseKey),
			},
			"SK": {
				S: aws.String(pollDatabaseKey),
			},
		},
	})
	if err != nil {
		log.Printf("error getting poll item: %s", err)

		return events.APIGatewayProxyResponse{
			Body:       http.StatusText(http.StatusNotFound),
			StatusCode: http.StatusNotFound,
		}, nil
	}

	if result.Item == nil {
		return events.APIGatewayProxyResponse{
			Body:       http.StatusText(http.StatusNotFound),
			StatusCode: http.StatusNotFound,
		}, nil
	}

	foundPoll := poll{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &foundPoll)
	if err != nil {
		log.Printf("error unmarshalling poll item: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	res, err := json.Marshal(getPollResponse{Data: foundPoll})
	if err != nil {
		log.Printf("error marshalling poll response: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(res),
		StatusCode: http.StatusOK,
	}, nil
}

func main() {
	lambda.Start(handler)
}
