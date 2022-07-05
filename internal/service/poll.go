package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexdunne/interactive-live-stream-poll-service/internal/repository"
)

var ErrRecordNotFound = errors.New("could not find record")

type Repo interface {
	GetPoll(ctx context.Context, pollID string) (repository.DatabasePoll, error)
	CreatePoll(ctx context.Context, poll repository.NewPoll) (repository.DatabasePoll, error)
	CreatePollVote(ctx context.Context, vote repository.NewPollVote) (repository.DatabasePollVote, error)
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

func (s *service) GetPoll(ctx context.Context, pollID string) (Poll, error) {
	poll, err := s.repo.GetPoll(ctx, pollID)
	if err != nil {
		if err == repository.ErrPollNotFound {
			return Poll{}, ErrRecordNotFound
		}

		return Poll{}, fmt.Errorf("getting poll: %w", err)
	}

	return mapDatabasePollToPoll(poll), nil
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

	return mapDatabasePollToPoll(newPoll), nil
}

func mapDatabasePollToPoll(dbPoll repository.DatabasePoll) Poll {
	var opts []PollOption
	for _, o := range dbPoll.Options {
		opts = append(opts, PollOption{
			ID:    o.ID,
			Label: o.Label,
		})
	}

	return Poll{
		ID:                   dbPoll.ID,
		Question:             dbPoll.Question,
		Options:              opts,
		ChannelARN:           dbPoll.ChannelARN,
		AggregatedVoteTotals: dbPoll.AggregatedVoteTotals,
	}
}
