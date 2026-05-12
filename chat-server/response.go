package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type HTTPResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, statusCode int, resp HTTPResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Println("write json response failed:", err)
	}
}

func writeSuccess(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, HTTPResponse{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, HTTPResponse{
		Code:    statusCode,
		Message: message,
	})
}

func sendWSMessage(client *Client, msg WsMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("marshal ws message failed:", err)
		return
	}

	if !sendWSData(client, data) {
		log.Println("send ws message skipped: client closed or buffer full")
	}
}

func sendWSData(client *Client, data []byte) (sent bool) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("send ws message failed:", err)
			sent = false
		}
	}()

	select {
	case <-client.Done:
		return false
	default:
	}

	select {
	case client.Send <- data:
		return true
	case <-client.Done:
		return false
	default:
		return false
	}
}

func sendWSError(client *Client, message string) {
	sendWSMessage(client, WsMessage{
		Type:    "error",
		Content: message,
	})
}
