package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store persists recent user/assistant pairs per session.
type Store struct {
	rdb      *redis.Client
	maxPairs int
	ttl      time.Duration
}

type pair struct {
	User      string `json:"user"`
	Assistant string `json:"assistant"`
}

// New creates a Redis-backed memory store.
func New(redisURL string, maxPairs int, ttl time.Duration) (*Store, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	if maxPairs < 1 {
		maxPairs = 10
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &Store{rdb: rdb, maxPairs: maxPairs, ttl: ttl}, nil
}

// Close closes the Redis client.
func (s *Store) Close() error {
	if s == nil || s.rdb == nil {
		return nil
	}
	return s.rdb.Close()
}

func key(sessionID string) string {
	return "session:" + sessionID
}

// Load returns a human-readable memory transcript.
func (s *Store) Load(ctx context.Context, sessionID string) (string, error) {
	if s == nil || sessionID == "" {
		return "", nil
	}
	raw, err := s.rdb.Get(ctx, key(sessionID)).Bytes()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var pairs []pair
	if err := json.Unmarshal(raw, &pairs); err != nil {
		return "", err
	}
	var out string
	for _, p := range pairs {
		out += "User: " + p.User + "\nAssistant: " + p.Assistant + "\n"
	}
	return out, nil
}

// Append adds a turn and trims to maxPairs.
func (s *Store) Append(ctx context.Context, sessionID, user, assistant string) error {
	if s == nil || sessionID == "" {
		return nil
	}
	var pairs []pair
	raw, err := s.rdb.Get(ctx, key(sessionID)).Bytes()
	if err != nil && err != redis.Nil {
		return err
	}
	if err == nil {
		_ = json.Unmarshal(raw, &pairs)
	}
	pairs = append(pairs, pair{User: user, Assistant: assistant})
	if len(pairs) > s.maxPairs {
		pairs = pairs[len(pairs)-s.maxPairs:]
	}
	b, err := json.Marshal(pairs)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, key(sessionID), b, s.ttl).Err()
}

// Nop is an in-memory no-op store for tests / when Redis is unavailable.
type Nop struct{}

func (Nop) Load(ctx context.Context, sessionID string) (string, error) { return "", nil }
func (Nop) Append(ctx context.Context, sessionID, user, assistant string) error {
	return nil
}
