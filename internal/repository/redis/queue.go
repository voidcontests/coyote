package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"github.com/voidcontests/coyote/internal/config"
	"github.com/voidcontests/coyote/internal/domain"
	"github.com/voidcontests/coyote/pkg/logger"
)

type MessageQueue struct {
	client *redis.Client
	sub    *redis.PubSub
}

func NewMessageQueue(c config.Redis) *MessageQueue {
	client := redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Password,
		DB:       c.DB,
	})

	return &MessageQueue{
		client: client,
	}
}

func (mq *MessageQueue) Subscribe(ctx context.Context, channel string) (<-chan domain.Submission, error) {
	mq.sub = mq.client.Subscribe(ctx, channel)

	_, err := mq.sub.Receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("subscribe to channel: %w", err)
	}

	submissionsChan := make(chan domain.Submission)
	redisChan := mq.sub.Channel()

	go func() {
		defer close(submissionsChan)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-redisChan:
				if !ok {
					return
				}

				var submission domain.Submission
				if err := json.Unmarshal([]byte(msg.Payload), &submission); err != nil {
					slog.Error("failed to unmarshal submission message", logger.Err(err))
					continue
				}

				select {
				case submissionsChan <- submission:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return submissionsChan, nil
}

func (mq *MessageQueue) Close() error {
	if mq.sub != nil {
		if err := mq.sub.Close(); err != nil {
			return err
		}
	}
	return mq.client.Close()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
