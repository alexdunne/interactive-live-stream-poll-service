package main

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandler(t *testing.T) {
	t.Run("Successful Request", func(t *testing.T) {
		_, err := handler(context.Background(), events.APIGatewayProxyRequest{
			PathParameters: map[string]string{
				"id": "abc_123",
			},
		})
		if err != nil {
			t.Fatal("Everything should be ok")
		}
	})
}
