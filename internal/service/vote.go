package service

import (
	"context"
	"encoding/json"
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

type broadcastPoll struct {
	ID                   string         `json:"id"`
	AggregatedVoteTotals map[string]int `json:"aggregatedVoteTotals"`
}

type broadcastUpdateUpdateMetadata struct {
	Data broadcastPoll `json:"data"`
}

func (s *service) IncrementPollTotals(ctx context.Context, pollID string, answerIncrements map[string]int) error {
	poll, err := s.repo.GetPoll(ctx, pollID)
	if err != nil {
		if err == repository.ErrPollNotFound {
			return fmt.Errorf("could not find the poll: %w", err)
		}

		return fmt.Errorf("getting poll: %w", err)
	}

	newTotals, err := s.repo.IncrementPollTotals(ctx, pollID, answerIncrements)
	if err != nil {
		return fmt.Errorf("incrementing totals: %w", err)
	}

	metadata := broadcastUpdateUpdateMetadata{
		Data: broadcastPoll{
			ID:                   poll.ID,
			AggregatedVoteTotals: newTotals,
		},
	}

	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("error marhsalling broadcast metadata: %w", err)
	}

	s.broadcaster.Broadcast(ctx, poll.ChannelARN, string(jsonMetadata))
	if err != nil {
		return fmt.Errorf("error sending metadata to channel: %w", err)
	}

	return nil
}

func mapDatabasePollVoteToVote(dbVote repository.DatabasePollVote) PollVote {
	return PollVote{
		ID:     dbVote.ID,
		PollID: dbVote.PollID,
		UserID: dbVote.UserID,
		Answer: dbVote.Answer,
	}
}
