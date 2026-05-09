package main

import (
	"context"
	"encoding/json"
	"fmt"
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

func SetOnline(userID int64, instanceID string, ttlSeconds int) error {
	key := fmt.Sprintf("online:%d", userID)
	return rdb.Set(
		context.Background(),
		key,
		instanceID,
		time.Duration(ttlSeconds)*time.Second,
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
