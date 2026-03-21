package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// Broadcaster syncs events across multiple orchestrator instances.
type Broadcaster struct {
	redis     *redis.Client
	manager   *Manager
	batcher   *Batcher
}

func NewBroadcaster(redis *redis.Client, manager *Manager, batcher *Batcher) *Broadcaster {
	return &Broadcaster{
		redis:   redis,
		manager: manager,
		batcher: batcher,
	}
}

// Publish local events to Dragonfly to reaching other instances.
func (b *Broadcaster) Publish(ctx context.Context, slug string, event interface{}) error {
	msg, err := json.Marshal(event)
	if err != nil {
		return err
	}
	
	channel := fmt.Sprintf("cascata:realtime:%s", slug)
	return b.redis.Publish(ctx, channel, msg).Err()
}

// RunBackgroundListener subscribes to Dragonfly to broadcast to local consumers.
func (b *Broadcaster) RunBackgroundListener(ctx context.Context) {
	// Pattern subscribe to all project channels
	pubsub := b.redis.PSubscribe(ctx, "cascata:realtime:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case msg := <-ch:
			// Extract slug from channel "cascata:realtime:PROJECT_SLUG"
			slug := msg.Channel[17:]
			
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}

			// Deliver to local SSE clients
			b.manager.Broadcast(slug, event)

		case <-ctx.Done():
			return
		}
	}
}
