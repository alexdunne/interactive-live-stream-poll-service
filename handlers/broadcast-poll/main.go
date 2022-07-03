package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/ivs"
)

const _pollMetdataType = "poll"

type poll struct {
	ID         string `json:"id"`
	ChannelARN string `json:"channelARN"`
}

type metadata struct {
	Type string `json:"type"`
	Data poll   `json:"data"`
}

func handle(ctx context.Context, event events.DynamoDBEvent) error {
	svc := ivs.New(session.Must(session.NewSession()))

	for _, record := range event.Records {
		var p poll
		if err := unmarshalStreamImage(record.Change.NewImage, &p); err != nil {
			log.Printf("error unmarshalling stream event into a poll: %s", err)
			continue
		}

		log.Printf("received a new poll %s for channel %s", p.ID, p.ChannelARN)

		pollMetadata := metadata{
			Type: _pollMetdataType,
			Data: p,
		}

		jsonMetadata, err := json.Marshal(pollMetadata)
		if err != nil {
			log.Printf("error marhsalling poll into metadata: %s", err)
			continue
		}

		_, err = svc.PutMetadataWithContext(ctx, &ivs.PutMetadataInput{
			ChannelArn: &p.ChannelARN,
			Metadata:   aws.String(string(jsonMetadata)),
		})
		if err != nil {
			log.Printf("error sending metadata to channel: %s", err)
			continue
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
