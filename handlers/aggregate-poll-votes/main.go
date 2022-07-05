package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/alexdunne/interactive-live-stream-poll-service/internal/repository"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/ivs"
)

const _tableNameEnv = "POLL_TABLE_NAME"

type TotalsPerPoll = map[string]map[string]int

type vote struct {
	PollID string `json:"pollId"`
	Answer string `json:"answer"`
}

type pollResult struct {
	ID     string         `json:"id"`
	Totals map[string]int `json:"totals"`
}

type metadata struct {
	Data pollResult `json:"data"`
}

func handle(ctx context.Context, event events.DynamoDBEvent) error {
	tableName, ok := os.LookupEnv(_tableNameEnv)
	if !ok {
		return fmt.Errorf("error environment variable %s not set", _tableNameEnv)
	}

	sess := session.Must(session.NewSession())

	db := dynamodb.New(sess)
	ivsSvc := ivs.New(sess)

	totalsPerPoll := aggregatedVoteTotals(event.Records)

	for pollID, totals := range totalsPerPoll {
		pollDatabaseID := fmt.Sprintf("POLL#%s", pollID)

		repo := repository.New(tableName, db)

		poll, err := repo.GetPoll(ctx, pollID)
		if err != nil {
			if err == repository.ErrPollNotFound {
				log.Printf("unable to find poll for id %s", pollID)
				continue
			}

			log.Printf("error getting poll item: %s", err)
			continue
		}

		type pollUpdate struct {
			updateExprParts []string
			attrNames       map[string]*string
			attrValues      map[string]*dynamodb.AttributeValue
		}

		update := pollUpdate{
			updateExprParts: []string{},
			attrNames:       make(map[string]*string),
			attrValues:      make(map[string]*dynamodb.AttributeValue),
		}
		index := 0

		// calculate the updates for the poll
		for answerID, total := range totals {
			attrName := fmt.Sprintf("#s%d", index)
			attrValKey := fmt.Sprintf(":val%d", index)

			update.updateExprParts = append(
				update.updateExprParts,
				fmt.Sprintf("aggregatedVoteTotal.%s = aggregatedVoteTotal.%s + :val%s", attrName, attrName, attrValKey),
			)

			if _, ok := update.attrNames[attrName]; !ok {
				update.attrNames[attrName] = aws.String(answerID)
			}
			if _, ok := update.attrValues[attrValKey]; !ok {
				update.attrValues[attrValKey] = &dynamodb.AttributeValue{
					N: aws.String(strconv.Itoa(total)),
				}
			}

			index = index + 1
		}

		updateExpr := fmt.Sprintf("set %s", strings.Join(update.updateExprParts, ","))

		input := &dynamodb.UpdateItemInput{
			TableName: aws.String(tableName),
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(pollDatabaseID),
				},
				"SK": {
					S: aws.String(pollDatabaseID),
				},
			},
			UpdateExpression:          aws.String(updateExpr),
			ExpressionAttributeNames:  update.attrNames,
			ExpressionAttributeValues: update.attrValues,
			ReturnValues:              aws.String("UPDATED_NEW"),
		}

		res, err := db.UpdateItemWithContext(ctx, input)
		if err != nil {
			log.Printf("error updating vote aggregate totals %s", err)
			continue
		}

		type updateItemResponse struct {
			AggregatedVoteTotals map[string]int `json:"aggregatedVoteTotals"`
		}

		var updateItemRes updateItemResponse
		if err := dynamodbattribute.UnmarshalMap(res.Attributes, &updateItemRes); err != nil {
			log.Printf("error unmarshalling update item output attributes %s", err)
			continue
		}

		totalsMetadata := metadata{
			Data: pollResult{
				ID:     poll.ID,
				Totals: updateItemRes.AggregatedVoteTotals,
			},
		}

		jsonMetadata, err := json.Marshal(totalsMetadata)
		if err != nil {
			log.Printf("error marhsalling totals into metadata: %s", err)
			continue
		}

		_, err = ivsSvc.PutMetadataWithContext(ctx, &ivs.PutMetadataInput{
			ChannelArn: &poll.ChannelARN,
			Metadata:   aws.String(string(jsonMetadata)),
		})
		if err != nil {
			log.Printf("error sending metadata to channel: %s", err)
			continue
		}
	}

	return nil
}

func aggregatedVoteTotals(records []events.DynamoDBEventRecord) TotalsPerPoll {
	totalsPerPoll := make(TotalsPerPoll)

	for _, record := range records {
		var v vote
		if err := unmarshalStreamImage(record.Change.NewImage, &v); err != nil {
			log.Printf("error unmarshalling stream event into a vote: %s", err)
			continue
		}

		if _, ok := totalsPerPoll[v.PollID]; !ok {
			totalsPerPoll[v.PollID] = make(map[string]int)
		}

		if _, ok := totalsPerPoll[v.PollID][v.Answer]; !ok {
			totalsPerPoll[v.PollID][v.Answer] = 0
		}

		totalsPerPoll[v.PollID][v.Answer] = totalsPerPoll[v.PollID][v.Answer] + 1
	}

	return totalsPerPoll
}

func unmarshalStreamImage(attribute map[string]events.DynamoDBAttributeValue, out interface{}) error {
	dbAttrMap := make(map[string]*dynamodb.AttributeValue)

	for k, v := range attribute {

		var dbAttr dynamodb.AttributeValue

		bytes, marshalErr := v.MarshalJSON()
		if marshalErr != nil {
			return marshalErr
		}

		json.Unmarshal(bytes, &dbAttr)
		dbAttrMap[k] = &dbAttr
	}

	return dynamodbattribute.UnmarshalMap(dbAttrMap, out)

}

func main() {
	lambda.Start(handle)
}
