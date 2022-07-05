package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
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
		return DatabasePollVote{}, fmt.Errorf("marshalling new vote: %w", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: r.tableName,
	}

	_, err = r.db.PutItem(input)
	if err != nil {
		return DatabasePollVote{}, fmt.Errorf("calling PutItem for vote: %w", err)
	}

	return dbVote, nil
}

type updateItemResponse struct {
	AggregatedVoteTotals DatabasePollTotals `json:"aggregatedVoteTotals"`
}

func (r *repo) IncrementPollTotals(ctx context.Context, pollID string, answerIncrements DatabasePollTotals) (DatabasePollTotals, error) {
	pollKey := buildPollDatabaseKey(pollID)

	input := r.getPollAnswerIncrementInput(pollKey, answerIncrements)

	res, err := r.db.UpdateItemWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("updating vote aggregate totals %w", err)
	}

	var updateItemRes updateItemResponse
	if err := dynamodbattribute.UnmarshalMap(res.Attributes, &updateItemRes); err != nil {
		return nil, fmt.Errorf("unmarshalling update item output attributes %w", err)

	}

	return updateItemRes.AggregatedVoteTotals, nil
}

func (r *repo) getPollAnswerIncrementInput(pollKey string, pollAnswerIncrements DatabasePollTotals) *dynamodb.UpdateItemInput {
	index := 0
	updateExprParts := []string{}
	attrNames := make(map[string]*string)
	attrValues := make(map[string]*dynamodb.AttributeValue)

	for answerID, incr := range pollAnswerIncrements {
		attrName := fmt.Sprintf("#s%d", index)
		attrValKey := fmt.Sprintf(":v%d", index)

		updateExprParts = append(
			updateExprParts,
			fmt.Sprintf("aggregatedVoteTotals.%s = aggregatedVoteTotals.%s + %s", attrName, attrName, attrValKey),
		)
		attrNames[attrName] = aws.String(answerID)
		attrValues[attrValKey] = &dynamodb.AttributeValue{
			N: aws.String(strconv.Itoa(incr)),
		}

		index = index + 1
	}

	updateExpr := fmt.Sprintf("set %s", strings.Join(updateExprParts, ","))

	return &dynamodb.UpdateItemInput{
		TableName: r.tableName,
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(pollKey),
			},
			"SK": {
				S: aws.String(pollKey),
			},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  attrNames,
		ExpressionAttributeValues: attrValues,
		ReturnValues:              aws.String("UPDATED_NEW"),
	}
}
