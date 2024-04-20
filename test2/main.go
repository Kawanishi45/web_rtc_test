package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var clients = make(map[*websocket.Conn]string) // 接続されたクライアントを管理
var broadcast = make(chan Message)             // ブロードキャストチャンネル

// WebSocketアップグレーダー
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type RTCSessionDescription struct {
	Type string `json:"type"`
	Sdp  string `json:"sdp"`
}

type Message struct {
	Type      string                 `json:"type"`
	Offer     *RTCSessionDescription `json:"offer,omitempty"`
	Answer    *RTCSessionDescription `json:"answer,omitempty"`
	Candidate string                 `json:"candidate,omitempty"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
}

func main() {
	http.HandleFunc("/ws", handleConnections)

	go handleMessages()

	fmt.Println("Server started on 0.0.0.0:8080")
	err := http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// WebSocket接続を確立
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer ws.Close()

	// クライアントを登録
	userIdStr := r.URL.Query().Get("userId")
	if userIdStr == "" {
		fmt.Println("Error getting userId")
		return
	}
	clients[ws] = userIdStr

	log.Println("\n\nClient connected:", r.URL.Query().Get("userId"))

	log.Println("\n\nClients:", clients)

	for {
		var msg Message
		// メッセージを受信
		err := ws.ReadJSON(&msg)
		if err != nil {
			fmt.Println("Error reading json:", err)
			log.Println(msg)
			delete(clients, ws)
			break
		}

		log.Println("\n\nReceived message:", msg)

		// 受信したメッセージをブロードキャストチャンネルに送信
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast

		// メッセージを適切なクライアントに送信
		for client, userId := range clients {
			log.Println("userId", userId)
			log.Println("msg.To", msg.To)
			if userId == msg.To {
				err := client.WriteJSON(msg)
				if err != nil {
					fmt.Println("Error sending message:", err)
					client.Close()
					delete(clients, client)
				}
				log.Println("\nSent message:", msg, "to", userId)
			}
		}
	}
}
