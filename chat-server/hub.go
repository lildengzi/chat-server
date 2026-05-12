package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait         = 10 * time.Second
	pongWait          = 60 * time.Second
	pingPeriod        = (pongWait * 9) / 10
	maxMessageSize    = 4096
	onlineTTL         = 120 * time.Second
	onlineRenewPeriod = onlineTTL / 2
)

type Client struct {
	UserID   int64
	Username string
	Send     chan []byte
	Conn     *websocket.Conn
	Done     chan struct{}
}

type Hub struct {
	clients    map[int64]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *WsMessage
	getOnline  chan chan []int64
	kick       chan int64
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int64]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *WsMessage, 256),
		getOnline:  make(chan chan []int64),
		kick:       make(chan int64),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			previousInstanceID, err := GetOnline(client.UserID)
			if err != nil {
				log.Println("get previous online failed:", err)
			}
			if oldClient, ok := h.clients[client.UserID]; ok {
				h.closeClient(oldClient)
			}
			h.clients[client.UserID] = client
			if err := SetOnline(client.UserID, InstanceID, onlineTTL); err != nil {
				log.Println("set online failed:", err)
			}
			if previousInstanceID != "" && previousInstanceID != InstanceID {
				if err := PublishKick(client.UserID, previousInstanceID); err != nil {
					log.Println("publish kick failed:", err)
				}
			}

		case client := <-h.unregister:
			if current, ok := h.clients[client.UserID]; ok && current == client {
				delete(h.clients, client.UserID)
				h.closeClient(client)
				if err := DelOnlineIfMatch(client.UserID, InstanceID); err != nil {
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

			if !sendWSData(client, data) {
				log.Println("drop message: client send buffer full")
			}

		case respChan := <-h.getOnline:
			ids := make([]int64, 0, len(h.clients))
			for uid := range h.clients {
				ids = append(ids, uid)
			}
			respChan <- ids

		case userID := <-h.kick:
			if client, ok := h.clients[userID]; ok {
				delete(h.clients, userID)
				h.closeClient(client)
			}
		}
	}
}

func (h *Hub) KickUser(userID int64) {
	h.kick <- userID
}

func (h *Hub) closeClient(client *Client) {
	select {
	case <-client.Done:
	default:
		close(client.Done)
		close(client.Send)
		_ = client.Conn.Close()
	}
}

func (h *Hub) GetOnlineUsers() []int64 {
	respChan := make(chan []int64)
	h.getOnline <- respChan
	return <-respChan
}
