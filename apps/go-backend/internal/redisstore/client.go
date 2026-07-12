package redisstore

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const LeaderboardKey = "leaderboard:journalist_rank"

func ArticleReadsKey(articleID string) string {
	return "article:" + articleID + ":reads"
}

func NewClient(url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return client, nil
}
