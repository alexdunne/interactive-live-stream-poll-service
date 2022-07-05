package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
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

type Poll struct {
	ID         string       `json:"id"`
	Question   string       `json:"question"`
	Options    []PollOption `json:"options"`
	ChannelARN string       `json:"channelARN"`
}

type PollOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func (r *repo) GetPoll(ctx context.Context, id string) (Poll, error) {
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
		return Poll{}, err
	}

	if result.Item == nil {
		return Poll{}, ErrPollNotFound
	}

	var foundPoll Poll
	err = dynamodbattribute.UnmarshalMap(result.Item, &foundPoll)
	if err != nil {
		return Poll{}, fmt.Errorf("error unmarshalling poll item: %w", err)
	}

	return foundPoll, nil
}

func buildPollDatabaseKey(id string) string {
	return fmt.Sprintf("POLL#%s", id)
}
