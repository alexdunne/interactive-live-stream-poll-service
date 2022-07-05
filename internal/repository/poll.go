package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

var ErrPollNotFound = errors.New("could not find poll")

type repo struct {
	tableName *string
	db        *dynamodb.DynamoDB
}

func New(tableName string, db *dynamodb.DynamoDB) *repo {
	return &repo{
		tableName: aws.String(tableName),
		db:        db,
	}
}

type DatabasePoll struct {
	PK                   string               `json:"PK"`
	SK                   string               `json:"SK"`
	ID                   string               `json:"id"`
	ItemType             string               `json:"itemType"`
	Question             string               `json:"question"`
	Options              []DatabasePollOption `json:"options"`
	ChannelARN           string               `json:"channelARN"`
	AggregatedVoteTotals DatabasePollTotals   `json:"aggregatedVoteTotals"`
}

type DatabasePollOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type DatabasePollTotals = map[string]int

func (r *repo) GetPoll(ctx context.Context, id string) (DatabasePoll, error) {
	pollKey := buildPollDatabaseKey(id)

	result, err := r.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: r.tableName,
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(pollKey),
			},
			"SK": {
				S: aws.String(pollKey),
			},
		},
	})
	if err != nil {
		return DatabasePoll{}, err
	}

	if result.Item == nil {
		return DatabasePoll{}, ErrPollNotFound
	}

	var foundPoll DatabasePoll
	err = dynamodbattribute.UnmarshalMap(result.Item, &foundPoll)
	if err != nil {
		return DatabasePoll{}, fmt.Errorf("error unmarshalling poll item: %w", err)
	}

	return foundPoll, nil
}

type NewPoll struct {
	Question   string
	Options    []string
	ChannelARN string
}

func (r *repo) CreatePoll(ctx context.Context, poll NewPoll) (DatabasePoll, error) {
	id := uuid.NewString()
	pollKey := buildPollDatabaseKey(id)

	var pollOptions []DatabasePollOption
	for _, o := range poll.Options {
		pollOptions = append(pollOptions, DatabasePollOption{
			ID:    uuid.NewString(),
			Label: o,
		})
	}

	totals := make(map[string]int, len(pollOptions))
	for _, o := range pollOptions {
		totals[o.ID] = 0
	}

	dbPoll := DatabasePoll{
		PK:                   pollKey,
		SK:                   pollKey,
		ID:                   id,
		ItemType:             "Poll",
		Question:             poll.Question,
		Options:              pollOptions,
		ChannelARN:           poll.ChannelARN,
		AggregatedVoteTotals: totals,
	}

	av, err := dynamodbattribute.MarshalMap(dbPoll)
	if err != nil {
		return DatabasePoll{}, fmt.Errorf("error marshalling new poll: %w", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: r.tableName,
	}

	_, err = r.db.PutItem(input)
	if err != nil {
		return DatabasePoll{}, fmt.Errorf("error calling PutItem for poll: %w", err)
	}

	return dbPoll, nil
}
