package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/alexdunne/interactive-live-stream-poll-service/internal/broadcast"
	"github.com/alexdunne/interactive-live-stream-poll-service/internal/utils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ivs"
)

type poll struct {
	ID         string `json:"id"`
	ChannelARN string `json:"channelARN"`
}

func handle(ctx context.Context, event events.DynamoDBEvent) error {
	svc := ivs.New(session.Must(session.NewSession()))
	broadcaster := broadcast.New(svc)

	for _, record := range event.Records {
		var p poll
		if err := utils.UnmarshalStreamImage(record.Change.NewImage, &p); err != nil {
			log.Printf("error unmarshalling stream event into a poll: %s", err)
			continue
		}

		log.Printf("received a new poll %s for channel %s", p.ID, p.ChannelARN)

		metadata := broadcast.CreateMetadata(p)

		jsonMetadata, err := json.Marshal(metadata)
		if err != nil {
			log.Printf("error marhsalling poll into metadata: %s", err)
			continue
		}

		if err := broadcaster.Broadcast(ctx, p.ChannelARN, string(jsonMetadata)); err != nil {
			log.Printf("error sending metadata to channel: %s", err)
			continue
		}
	}

	return nil
}

func main() {
	lambda.Start(handle)
}
