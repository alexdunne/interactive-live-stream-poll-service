package repository

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

type DatabasePollVote struct {
	PK       string `json:"PK"`
	SK       string `json:"SK"`
	ID       string `json:"id"`
	ItemType string `json:"itemType"`
	PollID   string `json:"pollId"`
	UserID   string `json:"userId"`
	Answer   string `json:"answer"`
}

type NewPollVote struct {
	PollID string
	UserID string
	Answer string
}

func (r *repo) CreatePollVote(ctx context.Context, v NewPollVote) (DatabasePollVote, error) {
	id := uuid.NewString()
	pollKey := buildPollDatabaseKey(v.PollID)
	userKey := buildUserDatabaseKey(v.UserID)

	dbVote := DatabasePollVote{
		PK:       pollKey,
		SK:       userKey,
		ID:       id,
		ItemType: "Vote",
		PollID:   v.PollID,
		UserID:   v.UserID,
		Answer:   v.Answer,
	}

	av, err := dynamodbattribute.MarshalMap(dbVote)
	if err != nil {
		return DatabasePollVote{}, fmt.Errorf("error marshalling new vote: %w", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: r.tableName,
	}

	_, err = r.db.PutItem(input)
	if err != nil {
		return DatabasePollVote{}, fmt.Errorf("error calling PutItem for vote: %w", err)
	}

	return dbVote, nil
}
