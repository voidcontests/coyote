package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/voidcontests/coyote/internal/domain"
	"github.com/voidcontests/coyote/internal/usecase/submission"
	"github.com/voidcontests/coyote/pkg/logger"
)

type Handler struct {
	ss *submission.Service
	mq domain.MessageQueue
}

func NewHandler(ss *submission.Service, mq domain.MessageQueue) *Handler {
	return &Handler{
		ss: ss,
		mq: mq,
	}
}

func (h *Handler) Listen(ctx context.Context, channel string) error {
	slog.Info("starting to listen for submissions", slog.String("channel", channel))

	submissionChan, err := h.mq.Subscribe(ctx, channel)
	if err != nil {
		return fmt.Errorf("subscribe to queue: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping submission listener")
			return ctx.Err()
		case submission, ok := <-submissionChan:
			if !ok {
				return fmt.Errorf("submission channel closed unexpectedly while listening for submissions on channel %q", channel)
			}

			slog.Debug("processing submission", slog.Int("submission_id", int(submission.ID)))

			err := h.ss.ProcessSubmission(ctx, submission)
			if err != nil {
				slog.Error("failed to process submission", slog.Int("submission_id", int(submission.ID)), logger.Err(err))
				continue
			}

			slog.Info("successfully processed submission", slog.Int("submission_id", int(submission.ID)))
		}
	}
}

func (h *Handler) Close() error {
	return h.mq.Close()
}
