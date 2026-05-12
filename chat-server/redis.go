package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func InitRedis(addr, password string, db int) error {
	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return rdb.Ping(context.Background()).Err()
}

func SetOnline(userID int64, instanceID string, ttl time.Duration) error {
	key := fmt.Sprintf("online:%d", userID)
	return rdb.Set(
		context.Background(),
		key,
		instanceID,
		ttl,
	).Err()
}

func GetOnline(userID int64) (string, error) {
	key := fmt.Sprintf("online:%d", userID)
	val, err := rdb.Get(context.Background(), key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

func DelOnline(userID int64) error {
	key := fmt.Sprintf("online:%d", userID)
	return rdb.Del(context.Background(), key).Err()
}

func DelOnlineIfMatch(userID int64, instanceID string) error {
	key := fmt.Sprintf("online:%d", userID)
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		end
		return 0
	`)
	return script.Run(context.Background(), rdb, []string{key}, instanceID).Err()
}

func PublishKick(userID int64, instanceID string) error {
	return Publish("kick_user", map[string]interface{}{
		"user_id":     userID,
		"instance_id": instanceID,
	})
}

func parseInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func Publish(channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return rdb.Publish(context.Background(), channel, string(data)).Err()
}

func Subscribe(channel string, handler func(string)) {
	go func() {
		pubsub := rdb.Subscribe(context.Background(), channel)
		defer pubsub.Close()

		ch := pubsub.Channel()
		for msg := range ch {
			handler(msg.Payload)
		}
	}()
}
