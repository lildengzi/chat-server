package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeWs(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	var user User
	err = dbPool.QueryRow(
		r.Context(),
		"SELECT user_id, username, password FROM users WHERE user_id=$1",
		userID,
	).Scan(&user.UserID, &user.Username, &user.Password)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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
	}

	hub.register <- client

	go writePump(client)
	go readPump(client)
}

func writePump(client *Client) {
	for data := range client.Send {
		if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}
	}
}

func readPump(client *Client) {
	defer func() {
		hub.unregister <- client.UserID
		client.Conn.Close()
	}()

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
		}
	}
}

func handleChat(client *Client, msg *WsMessage) {
	msg.FromUserID = client.UserID

	instanceID, err := GetOnline(msg.ToUserID)
	if err != nil {
		log.Println("get online failed:", err)
		return
	}

	if instanceID != "" {
		if err := Publish("chat_broadcast", msg); err != nil {
			log.Println("publish chat failed:", err)
		}
		return
	}

	if err := SaveOfflineMessage(msg.ToUserID, client.UserID, msg.Content); err != nil {
		log.Println("save offline message failed:", err)
	}
}

func handleGetOnlineList(client *Client) {
	userIDs := hub.GetOnlineUsers()

	resp := WsMessage{
		Type: "online_list",
		Data: userIDs,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Println("marshal online list failed:", err)
		return
	}

	client.Send <- data
}

func handleGetOffline(client *Client) {
	messages, err := FetchAndDeleteOfflineMessages(client.UserID)
	if err != nil {
		log.Println("fetch offline messages failed:", err)
		return
	}

	resp := WsMessage{
		Type: "offline_list",
		Data: messages,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Println("marshal offline list failed:", err)
		return
	}

	client.Send <- data
}

func handleRemoteMessage(payload string) {
	var msg WsMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		log.Println("unmarshal remote message failed:", err)
		return
	}

	hub.broadcast <- &msg
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	userID, err := CreateUser(req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int64{"user_id": userID})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, err := GetUserByUsername(req.Username, req.Password)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}
