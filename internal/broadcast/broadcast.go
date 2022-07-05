package broadcast

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ivs"
)

type MetadataPutter interface {
	PutMetadataWithContext(ctx context.Context, input *ivs.PutMetadataInput, opts ...request.Option) (*ivs.PutMetadataOutput, error)
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
	_, err := s.broadcaster.PutMetadataWithContext(ctx, &ivs.PutMetadataInput{
		ChannelArn: &channelARN,
		Metadata:   &data,
	})

	return err
}
