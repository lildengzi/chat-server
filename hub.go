package main

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserID   int64
	Username string
	Send     chan []byte
	Conn     *websocket.Conn
}

type Hub struct {
	clients    map[int64]*Client
	register   chan *Client
	unregister chan int64
	broadcast  chan *WsMessage
	getOnline  chan chan []int64
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int64]*Client),
		register:   make(chan *Client),
		unregister: make(chan int64),
		broadcast:  make(chan *WsMessage, 256),
		getOnline:  make(chan chan []int64),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.UserID] = client
			if err := SetOnline(client.UserID, InstanceID, 120); err != nil {
				log.Println("set online failed:", err)
			}

		case userID := <-h.unregister:
			if client, ok := h.clients[userID]; ok {
				delete(h.clients, userID)
				close(client.Send)
				if err := DelOnline(userID); err != nil {
					log.Println("delete online failed:", err)
				}
			}

		case msg := <-h.broadcast:
			client, ok := h.clients[msg.ToUserID]
			if !ok {
				continue
			}

			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}

			client.Send <- data

		case respChan := <-h.getOnline:
			ids := make([]int64, 0, len(h.clients))
			for uid := range h.clients {
				ids = append(ids, uid)
			}
			respChan <- ids
		}
	}
}

func (h *Hub) GetOnlineUsers() []int64 {
	respChan := make(chan []int64)
	h.getOnline <- respChan
	return <-respChan
}
