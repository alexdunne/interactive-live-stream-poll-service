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

type pollOverview struct {
	ID                   string         `json:"id"`
	Question             string         `json:"question"`
	Options              []pollOption   `json:"options"`
	AggregatedVoteTotals map[string]int `json:"aggregatedVoteTotals"`
}

type pollOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type getPollResponse struct {
	Data pollOverview `json:"data"`
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

	poll, err := svc.GetPoll(ctx, pollID)
	if err != nil {
		if err == service.ErrRecordNotFound {
			return api.ClientError(http.StatusNotFound, "poll not found")
		}

		return api.ServerError(fmt.Errorf("error getting poll item: %s", err))
	}

	res, err := json.Marshal(mapPollToResponse(poll))
	if err != nil {
		return api.ServerError(fmt.Errorf("error marshalling poll response: %s", err))
	}

	return events.APIGatewayProxyResponse{
		Body:       string(res),
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
	}, nil
}

func mapPollToResponse(p service.Poll) getPollResponse {
	var po []pollOption
	for _, opt := range p.Options {
		po = append(po, pollOption{
			ID:    opt.ID,
			Label: opt.Label,
		})
	}

	return getPollResponse{
		Data: pollOverview{
			ID:                   p.ID,
			Question:             p.Question,
			Options:              po,
			AggregatedVoteTotals: p.AggregatedVoteTotals,
		},
	}
}

func main() {
	lambda.Start(handler)
}
