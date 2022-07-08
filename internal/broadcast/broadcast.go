package broadcast

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ivs"
)

type MetadataPutter interface {
	PutMetadata(ctx context.Context, params *ivs.PutMetadataInput, optFns ...func(*ivs.Options)) (*ivs.PutMetadataOutput, error)
}

type service struct {
	broadcaster MetadataPutter
}

func New(mp MetadataPutter) *service {
	return &service{
		broadcaster: mp,
	}
}

func (s *service) Broadcast(ctx context.Context, channelARN string, data string) error {
	_, err := s.broadcaster.PutMetadata(ctx, &ivs.PutMetadataInput{
		ChannelArn: &channelARN,
		Metadata:   &data,
	})

	return err
}
