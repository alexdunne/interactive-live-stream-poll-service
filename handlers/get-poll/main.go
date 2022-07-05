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
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ivs"
)

const _tableNameEnv = "POLL_TABLE_NAME"

type pollOverview struct {
	ID       string       `json:"id"`
	Question string       `json:"question"`
	Options  []pollOption `json:"options"`
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

	poll, err := svc.GetPoll(ctx, pollID)
	if err != nil {
		if err == service.ErrRecordNotFound {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusNotFound,
				Body:       http.StatusText(http.StatusNotFound),
			}, nil
		}

		log.Printf("error getting poll item: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	res, err := json.Marshal(mapPollToResponse(poll))
	if err != nil {
		log.Printf("error marshalling poll response: %s", err)
		return api.InternalServerErrorResponse(), nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(res),
		StatusCode: http.StatusOK,
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
			ID:       p.ID,
			Question: p.Question,
			Options:  po,
		},
	}
}

func main() {
	lambda.Start(handler)
}
