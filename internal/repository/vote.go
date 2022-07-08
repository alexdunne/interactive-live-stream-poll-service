package repository

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type DatabasePollVote struct {
	PK       string `dynamodbav:"PK"`
	SK       string `dynamodbav:"SK"`
	ID       string `dynamodbav:"id"`
	ItemType string `dynamodbav:"itemType"`
	PollID   string `dynamodbav:"pollId"`
	UserID   string `dynamodbav:"userId"`
	Answer   string `dynamodbav:"answer"`
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

	item, err := attributevalue.MarshalMap(dbVote)
	if err != nil {
		return DatabasePollVote{}, fmt.Errorf("marshalling new povotell: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: r.tableName,
		Item:      item,
	}

	_, err = r.db.PutItem(ctx, input)
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

	input, err := r.getPollAnswerIncrementInput(pollKey, answerIncrements)
	if err != nil {
		return nil, fmt.Errorf("creating increment update input %w", err)
	}

	res, err := r.db.UpdateItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("updating vote aggregate totals %w", err)
	}

	var updateItemRes updateItemResponse
	if err := attributevalue.UnmarshalMap(res.Attributes, &updateItemRes); err != nil {
		return nil, fmt.Errorf("unmarshalling update item output attributes %w", err)

	}

	return updateItemRes.AggregatedVoteTotals, nil
}

func (r *repo) getPollAnswerIncrementInput(pollKey string, pollAnswerIncrements DatabasePollTotals) (*dynamodb.UpdateItemInput, error) {
	builder := expression.UpdateBuilder{}

	for answerID, incr := range pollAnswerIncrements {
		attrName := fmt.Sprintf("aggregatedVoteTotals.%s", answerID)

		builder = builder.Set(
			expression.Name(attrName),
			expression.Value(fmt.Sprintf("%s + %d", attrName, incr)),
		)
	}

	expr, err := expression.NewBuilder().WithUpdate(builder).Build()
	if err != nil {
		return nil, fmt.Errorf("building expression: %w", err)
	}

	return &dynamodb.UpdateItemInput{
		TableName: r.tableName,
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{
				Value: pollKey,
			},
			"SK": &types.AttributeValueMemberS{
				Value: pollKey,
			},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ConditionExpression:       expr.Condition(),
		ReturnValues:              types.ReturnValueUpdatedNew,
	}, nil
}
