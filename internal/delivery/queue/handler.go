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
	submissionService *submission.Service
	messageQueue      domain.MessageQueue
}

func NewHandler(submissionService *submission.Service, messageQueue domain.MessageQueue) *Handler {
	return &Handler{
		submissionService: submissionService,
		messageQueue:      messageQueue,
	}
}

func (h *Handler) Listen(ctx context.Context, channel string) error {
	slog.Info("starting to listen for submissions", slog.String("channel", channel))

	submissionChan, err := h.messageQueue.Subscribe(ctx, channel)
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

			err := h.submissionService.ProcessSubmission(ctx, submission)
			if err != nil {
				slog.Error("failed to process submission", slog.Int("submission_id", int(submission.ID)), logger.Err(err))
				continue
			}

			slog.Info("successfully processed submission", slog.Int("submission_id", int(submission.ID)))
		}
	}
}

func (h *Handler) Close() error {
	return h.messageQueue.Close()
}
