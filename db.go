package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool

func InitDB(connString string) error {
	var err error
	dbPool, err = pgxpool.New(context.Background(), connString)
	if err != nil {
		return err
	}
	return dbPool.Ping(context.Background())
}

func GetUserByUsername(username, password string) (*User, error) {
	var user User
	err := dbPool.QueryRow(
		context.Background(),
		"SELECT user_id, username, password FROM users WHERE username=$1 AND password=$2",
		username,
		password,
	).Scan(&user.UserID, &user.Username, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func CreateUser(username, password string) (int64, error) {
	var userID int64
	err := dbPool.QueryRow(
		context.Background(),
		"INSERT INTO users (username, password) VALUES ($1, $2) RETURNING user_id",
		username,
		password,
	).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func AddFriend(userID, friendID int64) error {
	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(
		context.Background(),
		"INSERT INTO friends (user_id, friend_id) VALUES ($1, $2), ($2, $1)",
		userID,
		friendID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(context.Background())
}

func GetFriends(userID int64) ([]int64, error) {
	rows, err := dbPool.Query(
		context.Background(),
		"SELECT friend_id FROM friends WHERE user_id=$1",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friendIDs []int64
	for rows.Next() {
		var friendID int64
		if err := rows.Scan(&friendID); err != nil {
			return nil, err
		}
		friendIDs = append(friendIDs, friendID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return friendIDs, nil
}

func SaveOfflineMessage(toUserID, fromUserID int64, content string) error {
	_, err := dbPool.Exec(
		context.Background(),
		"INSERT INTO offline_messages (to_user_id, from_user_id, content) VALUES ($1, $2, $3)",
		toUserID,
		fromUserID,
		content,
	)
	return err
}

func FetchAndDeleteOfflineMessages(userID int64) ([]OfflineMessage, error) {
	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())

	rows, err := tx.Query(
		context.Background(),
		"SELECT msg_id, to_user_id, from_user_id, content, created_at FROM offline_messages WHERE to_user_id=$1 ORDER BY created_at",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []OfflineMessage
	for rows.Next() {
		var msg OfflineMessage
		if err := rows.Scan(&msg.MsgID, &msg.ToUserID, &msg.FromUserID, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	_, err = tx.Exec(
		context.Background(),
		"DELETE FROM offline_messages WHERE to_user_id=$1",
		userID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}

	return messages, nil
}
