package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

const _tableNameEnv = "POLL_TABLE_NAME"

type vote struct {
	PollID string `json:"pollId"`
	Answer string `json:"answer"`
}

func handle(ctx context.Context, event events.DynamoDBEvent) error {
	tableName, ok := os.LookupEnv(_tableNameEnv)
	if !ok {
		return fmt.Errorf("error environment variable %s not set", _tableNameEnv)
	}

	db := dynamodb.New(session.Must(session.NewSession()))

	totalsPerPoll := make(map[string]map[string]int)

	for _, record := range event.Records {
		var v vote
		if err := unmarshalStreamImage(record.Change.NewImage, &v); err != nil {
			log.Printf("error unmarshalling stream event into a vote: %s", err)
			continue
		}

		log.Printf("received a new vote for poll %s", v.PollID)

		if _, ok := totalsPerPoll[v.PollID]; !ok {
			totalsPerPoll[v.PollID] = make(map[string]int)
		}

		if _, ok := totalsPerPoll[v.PollID][v.Answer]; !ok {
			totalsPerPoll[v.PollID][v.Answer] = 0
		}

		totalsPerPoll[v.PollID][v.Answer] = totalsPerPoll[v.PollID][v.Answer] + 1
	}

	for pollID, totals := range totalsPerPoll {
		pollDatabaseID := fmt.Sprintf("POLL#%s", pollID)

		// being lazy and doing multiple update requests rather than batching

		for answerID, total := range totals {
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
				UpdateExpression: aws.String("set aggregatedVoteTotal.#s= :val"),
				ExpressionAttributeNames: map[string]*string{
					"#s": aws.String(answerID),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":val": {
						N: aws.String(strconv.Itoa(total)),
					},
				},
			}

			_, err := db.UpdateItemWithContext(ctx, input)
			if err != nil {
				log.Printf("error updating vote aggregate totals %s", err)
				continue
			}
		}
	}

	return nil
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
