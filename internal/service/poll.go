package service

import (
	"context"
	"fmt"

	"github.com/alexdunne/interactive-live-stream-poll-service/internal/repository"
)

type Repo interface {
	CreatePoll(ctx context.Context, poll repository.NewPoll) (repository.DatabasePoll, error)
}

type Poll struct {
	ID                   string
	Question             string
	Options              []PollOption
	ChannelARN           string
	AggregatedVoteTotals map[string]int
}

type PollOption struct {
	ID    string
	Label string
}

type service struct {
	repo Repo
}

func New(r Repo) *service {
	return &service{
		repo: r,
	}
}

type NewPoll struct {
	Question   string
	Options    []string
	ChannelARN string
}

func (s *service) CreatePoll(ctx context.Context, poll NewPoll) (Poll, error) {
	newPoll, err := s.repo.CreatePoll(ctx, repository.NewPoll{
		Question:   poll.Question,
		Options:    poll.Options,
		ChannelARN: poll.ChannelARN,
	})
	if err != nil {
		return Poll{}, fmt.Errorf("creating new poll: %w", err)
	}

	var opts []PollOption
	for _, o := range newPoll.Options {
		opts = append(opts, PollOption{
			ID:    o.ID,
			Label: o.Label,
		})
	}

	return Poll{
		ID:                   newPoll.ID,
		Question:             newPoll.Question,
		Options:              opts,
		ChannelARN:           newPoll.ChannelARN,
		AggregatedVoteTotals: newPoll.AggregatedVoteTotals,
	}, nil
}
