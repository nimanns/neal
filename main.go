package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Game struct {
	Players map[string]*Player
	mu      sync.RWMutex
}

type Player struct {
	ID       string  `json:"id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Velocity struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"velocity"`
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

var (
	game     = &Game{Players: make(map[string]*Player)}
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all connections for development
		},
	}
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	playerID := generatePlayerID()
	
	player := &Player{
		ID: playerID,
		X:  100,
		Y:  100,
	}

	game.mu.Lock()
	game.Players[playerID] = player
	game.mu.Unlock()

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}

		var message Message
		if err := json.Unmarshal(p, &message); err != nil {
			log.Println("Failed to unmarshal message:", err)
			continue
		}

		switch message.Type {
		case "move":
			handleMoveMessage(playerID, message.Payload)
		}

		broadcastGameState(conn)
	}

	game.mu.Lock()
	delete(game.Players, playerID)
	game.mu.Unlock()
}

func handleMoveMessage(playerID string, payload interface{}) {
	game.mu.Lock()
	defer game.mu.Unlock()

	if player, ok := game.Players[playerID]; ok {
		if moveData, ok := payload.(map[string]interface{}); ok {
			if dx, ok := moveData["dx"].(float64); ok {
				player.X += dx
			}
			if dy, ok := moveData["dy"].(float64); ok {
				player.Y += dy
			}
		}
	}
}

func broadcastGameState(conn *websocket.Conn) {
	game.mu.RLock()
	gameState := game.Players
	game.mu.RUnlock()

	message := Message{
		Type:    "gameState",
		Payload: gameState,
	}

	if err := conn.WriteJSON(message); err != nil {
		log.Println("Failed to broadcast game state:", err)
	}
}

func generatePlayerID() string {
	game.mu.Lock()
	defer game.mu.Unlock()
	return fmt.Sprintf("player_%d", len(game.Players)+1)
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
