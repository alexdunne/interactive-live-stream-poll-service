package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/uuid"
)

var ErrPollNotFound = errors.New("could not find poll")

type repo struct {
	tableName *string
	db        *dynamodb.Client
}

func New(tableName string, db *dynamodb.Client) *repo {
	return &repo{
		tableName: aws.String(tableName),
		db:        db,
	}
}

type DatabasePoll struct {
	PK                   string               `dynamodbav:"PK"`
	SK                   string               `dynamodbav:"SK"`
	ID                   string               `dynamodbav:"id"`
	ItemType             string               `dynamodbav:"itemType"`
	Question             string               `dynamodbav:"question"`
	Options              []DatabasePollOption `dynamodbav:"options"`
	ChannelARN           string               `dynamodbav:"channelARN"`
	AggregatedVoteTotals DatabasePollTotals   `dynamodbav:"aggregatedVoteTotals"`
}

type DatabasePollOption struct {
	ID    string `dynamodbav:"id"`
	Label string `dynamodbav:"label"`
}

type DatabasePollTotals = map[string]int

func (r *repo) GetPoll(ctx context.Context, id string) (DatabasePoll, error) {
	pollKey := buildPollDatabaseKey(id)

	input := &dynamodb.GetItemInput{
		TableName: r.tableName,
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{
				Value: pollKey,
			},
			"SK": &types.AttributeValueMemberS{
				Value: pollKey,
			},
		},
	}

	result, err := r.db.GetItem(ctx, input)
	if err != nil {
		return DatabasePoll{}, err
	}

	if result.Item == nil {
		return DatabasePoll{}, ErrPollNotFound
	}

	var foundPoll DatabasePoll
	err = attributevalue.UnmarshalMap(result.Item, &foundPoll)
	if err != nil {
		return DatabasePoll{}, fmt.Errorf("unmarshalling poll item: %w", err)
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

	item, err := attributevalue.MarshalMap(dbPoll)
	if err != nil {
		return DatabasePoll{}, fmt.Errorf("marshalling new poll: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: r.tableName,
		Item:      item,
	}

	_, err = r.db.PutItem(ctx, input)
	if err != nil {
		return DatabasePoll{}, fmt.Errorf("calling PutItem for poll: %w", err)
	}

	return dbPoll, nil
}
