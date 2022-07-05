package service

import (
	"context"
	"fmt"

	"github.com/alexdunne/interactive-live-stream-poll-service/internal/repository"
)

type PollVote struct {
	ID     string
	PollID string
	UserID string
	Answer string
}

type NewPollVote struct {
	PollID string
	UserID string
	Answer string
}

func (s *service) CreatePollVote(ctx context.Context, v NewPollVote) (PollVote, error) {
	newVote, err := s.repo.CreatePollVote(ctx, repository.NewPollVote{
		PollID: v.PollID,
		UserID: v.UserID,
		Answer: v.Answer,
	})
	if err != nil {
		return PollVote{}, fmt.Errorf("creating new poll vote: %w", err)
	}

	return mapDatabasePollVoteToVote(newVote), nil
}

func mapDatabasePollVoteToVote(dbVote repository.DatabasePollVote) PollVote {
	return PollVote{
		ID:     dbVote.ID,
		PollID: dbVote.PollID,
		UserID: dbVote.UserID,
		Answer: dbVote.Answer,
	}
}
