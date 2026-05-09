package main

import "time"

type User struct {
	UserID   int64  `json:"user_id" db:"user_id"`
	Username string `json:"username" db:"username"`
	Password string `json:"password" db:"password"`
}

type Friend struct {
	UserID   int64 `json:"user_id" db:"user_id"`
	FriendID int64 `json:"friend_id" db:"friend_id"`
}

type OfflineMessage struct {
	MsgID      int64     `json:"msg_id" db:"msg_id"`
	ToUserID   int64     `json:"to_user_id" db:"to_user_id"`
	FromUserID int64     `json:"from_user_id" db:"from_user_id"`
	Content    string    `json:"content" db:"content"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type WsMessage struct {
	Type       string      `json:"type"`
	ToUserID   int64       `json:"to_user_id,omitempty"`
	FromUserID int64       `json:"from_user_id,omitempty"`
	Content    string      `json:"content,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}
