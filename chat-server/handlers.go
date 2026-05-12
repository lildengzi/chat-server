package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeWs(w http.ResponseWriter, r *http.Request) {
	token := GetBearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}

	claims, err := ParseToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := GetUserByID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		UserID:   user.UserID,
		Username: user.Username,
		Send:     make(chan []byte, 256),
		Conn:     conn,
		Done:     make(chan struct{}),
	}

	hub.register <- client

	go writePump(client)
	go readPump(client)
	go renewOnline(client)
}

func writePump(client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case data, ok := <-client.Send:
			if err := client.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				_ = client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			if err := client.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-client.Done:
			return
		}
	}
}

func readPump(client *Client) {
	defer func() {
		hub.unregister <- client
	}()

	client.Conn.SetReadLimit(maxMessageSize)
	if err := client.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return
	}
	client.Conn.SetPongHandler(func(string) error {
		return client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var msg WsMessage
		if err := client.Conn.ReadJSON(&msg); err != nil {
			return
		}

		switch msg.Type {
		case "chat":
			handleChat(client, &msg)
		case "get_online_list":
			handleGetOnlineList(client)
		case "get_offline":
			handleGetOffline(client)
		default:
			sendWSError(client, "unsupported message type")
		}
	}
}

func renewOnline(client *Client) {
	ticker := time.NewTicker(onlineRenewPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := SetOnline(client.UserID, InstanceID, onlineTTL); err != nil {
				log.Println("renew online failed:", err)
			}
		case <-client.Done:
			return
		}
	}
}

func handleChat(client *Client, msg *WsMessage) {
	msg.Content = strings.TrimSpace(msg.Content)
	if msg.ToUserID <= 0 {
		sendWSError(client, "invalid to_user_id")
		return
	}
	if msg.Content == "" {
		sendWSError(client, "content is required")
		return
	}
	if !userExists(msg.ToUserID) {
		sendWSError(client, "target user not found")
		return
	}

	msg.FromUserID = client.UserID

	instanceID, err := GetOnline(msg.ToUserID)
	if err != nil {
		log.Println("get online failed:", err)
		sendWSError(client, "get online status failed")
		return
	}

	if instanceID != "" {
		if err := Publish("chat_broadcast", msg); err != nil {
			log.Println("publish chat failed:", err)
			sendWSError(client, "send message failed")
		}
		return
	}

	if err := SaveOfflineMessage(msg.ToUserID, client.UserID, msg.Content); err != nil {
		log.Println("save offline message failed:", err)
		sendWSError(client, "save offline message failed")
	}
}

func handleGetOnlineList(client *Client) {
	userIDs := hub.GetOnlineUsers()

	resp := WsMessage{
		Type: "online_list",
		Data: userIDs,
	}

	sendWSMessage(client, resp)
}

func handleGetOffline(client *Client) {
	messages, err := FetchAndDeleteOfflineMessages(client.UserID)
	if err != nil {
		log.Println("fetch offline messages failed:", err)
		sendWSError(client, "fetch offline messages failed")
		return
	}

	resp := WsMessage{
		Type: "offline_list",
		Data: messages,
	}

	sendWSMessage(client, resp)
}

func handleRemoteMessage(payload string) {
	var msg WsMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		log.Println("unmarshal remote message failed:", err)
		return
	}

	hub.broadcast <- &msg
}

func handleKickMessage(payload string) {
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		log.Println("unmarshal kick message failed:", err)
		return
	}

	instanceID, ok := msg["instance_id"].(string)
	if !ok || instanceID != InstanceID {
		return
	}

	userID, ok := parseInt64(msg["user_id"])
	if !ok {
		log.Println("invalid kick user_id")
		return
	}

	hub.KickUser(userID)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	userID, err := CreateUser(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeSuccess(w, map[string]int64{"user_id": userID})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	user, err := CheckUserPassword(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	token, err := GenerateToken(user)
	if err != nil {
		log.Println("generate token failed:", err)
		writeError(w, http.StatusInternalServerError, "generate token failed")
		return
	}

	writeSuccess(w, LoginResponse{
		UserID:   user.UserID,
		Username: user.Username,
		Token:    token,
	})
}

func userExists(userID int64) bool {
	var exists bool
	err := dbPool.QueryRow(
		context.Background(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE user_id=$1)",
		userID,
	).Scan(&exists)
	if err != nil {
		log.Println("check user exists failed:", err)
		return false
	}
	return exists
}
