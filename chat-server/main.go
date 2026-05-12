package main

import (
	"log"
	"net/http"
	"os"
)

var InstanceID = getEnv("INSTANCE_ID", "server-1")
var hub = NewHub()

func main() {
	dbURL := os.Getenv("DB_URL")
	if err := InitDB(dbURL); err != nil {
		log.Fatal(err)
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	if err := InitRedis(redisAddr, "", 0); err != nil {
		log.Fatal(err)
	}

	go hub.Run()
	Subscribe("chat_broadcast", handleRemoteMessage)
	Subscribe("kick_user", handleKickMessage)

	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/ws", ServeWs)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
