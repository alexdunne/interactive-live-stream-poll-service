package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/alexdunne/interactive-live-stream-poll-service/internal/broadcast"
	"github.com/alexdunne/interactive-live-stream-poll-service/internal/repository"
	"github.com/alexdunne/interactive-live-stream-poll-service/internal/service"
	"github.com/alexdunne/interactive-live-stream-poll-service/internal/utils"
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

type TotalsPerPoll = map[string]map[string]int

type incomingVote struct {
	PollID string `json:"pollId"`
	Answer string `json:"answer"`
}

func handle(ctx context.Context, event events.DynamoDBEvent) error {
	tableName, ok := os.LookupEnv(_tableNameEnv)
	if !ok {
		return fmt.Errorf("error environment variable %s not set", _tableNameEnv)
	}

	repo := repository.New(tableName, &db)
	broadcaster := broadcast.New(&ivsClient)

	svc := service.New(repo, broadcaster)

	votesPerPoll := splitVotesByPollID(event.Records)

	for pollID, votes := range votesPerPoll {
		answerTotals := aggregatePollVoteTotals(votes)

		if err := svc.IncrementPollTotals(ctx, pollID, answerTotals); err != nil {
			log.Printf("error incrementing poll totals: %s", err)
		}
	}

	return nil
}

// splitVotesByPollID loops over the incoming events and splits them by their poll ID
func splitVotesByPollID(records []events.DynamoDBEventRecord) map[string][]incomingVote {
	eventsPerPoll := make(map[string][]incomingVote)

	for _, record := range records {
		var v incomingVote
		if err := utils.UnmarshalStreamImage(record.Change.NewImage, &v); err != nil {
			log.Printf("error unmarshalling stream event into a vote: %s", err)
			continue
		}

		if _, ok := eventsPerPoll[v.PollID]; !ok {
			eventsPerPoll[v.PollID] = make([]incomingVote, 0)
		}
		eventsPerPoll[v.PollID] = append(eventsPerPoll[v.PollID], v)
	}

	return eventsPerPoll
}

func aggregatePollVoteTotals(votes []incomingVote) map[string]int {
	aggregateTotals := make(map[string]int)

	for _, v := range votes {
		if _, ok := aggregateTotals[v.Answer]; !ok {
			aggregateTotals[v.Answer] = 0
		}

		aggregateTotals[v.Answer] = aggregateTotals[v.Answer] + 1
	}

	return aggregateTotals
}

func main() {
	lambda.Start(handle)
}
