package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	rateLimitWindow = 60 * time.Second
	rateLimitMax    = 20
)

// RateLimiter enforces a per-actor sliding-ish window using Redis INCR+EXPIRE.
type RateLimiter struct {
	rdb *redis.Client
}

func newRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

// Allow increments the counter for actorID and reports whether the caller
// is still within the allowed rate (count <= rateLimitMax) for this window.
func (r *RateLimiter) Allow(ctx context.Context, actorID string) (bool, error) {
	key := fmt.Sprintf("ratelimit:%s", actorID)
	count, err := r.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		if err := r.rdb.Expire(ctx, key, rateLimitWindow).Err(); err != nil {
			return false, err
		}
	}
	return count <= rateLimitMax, nil
}

// publishToConversation publishes a raw event payload to the conv:{id}
// pub/sub channel so any other connection watching the same conversation
// can pick it up.
func publishToConversation(ctx context.Context, rdb *redis.Client, conversationID string, payload []byte) error {
	channel := fmt.Sprintf("conv:%s", conversationID)
	return rdb.Publish(ctx, channel, payload).Err()
}
