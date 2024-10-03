package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Game struct {
	Players       map[string]*Player
	Collectibles  map[string]*Collectible
	mu            sync.RWMutex
	WorldWidth    float64
	WorldHeight   float64
	CollectibleID int
}

type Player struct {
	ID       string  `json:"id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Score    int     `json:"score"`
	Radius   float64 `json:"radius"`
	Color    string  `json:"color"`
	Velocity struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"velocity"`
}

type Collectible struct {
	ID     string  `json:"id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
	Color  string  `json:"color"`
	Points int     `json:"points"`
}

type GameState struct {
	Players      map[string]*Player     `json:"players"`
	Collectibles map[string]*Collectible `json:"collectibles"`
	WorldWidth   float64                `json:"worldWidth"`
	WorldHeight  float64                `json:"worldHeight"`
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

var (
	game = &Game{
		Players:      make(map[string]*Player),
		Collectibles: make(map[string]*Collectible),
		WorldWidth:   800,
		WorldHeight:  600,
	}
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	colors = []string{"#1E90FF", "#FF69B4", "#32CD32", "#FFD700", "#FF4500"}
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	playerID := generatePlayerID()
	player := createNewPlayer(playerID)

	game.mu.Lock()
	game.Players[playerID] = player
	game.mu.Unlock()

	sendGameState(conn)

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

		checkCollisions(playerID)
		sendGameState(conn)
	}

	game.mu.Lock()
	delete(game.Players, playerID)
	game.mu.Unlock()
}

func createNewPlayer(id string) *Player {
	colorIdx := rand.Intn(len(colors))
	return &Player{
		ID:     id,
		X:      rand.Float64() * game.WorldWidth,
		Y:      rand.Float64() * game.WorldHeight,
		Radius: 20,
		Color:  colors[colorIdx],
		Score:  0,
	}
}

func handleMoveMessage(playerID string, payload interface{}) {
	game.mu.Lock()
	defer game.mu.Unlock()

	player, ok := game.Players[playerID]
	if !ok {
		return
	}

	if moveData, ok := payload.(map[string]interface{}); ok {
		if dx, ok := moveData["dx"].(float64); ok {
			newX := player.X + dx
			if newX >= 0 && newX <= game.WorldWidth {
				player.X = newX
			}
		}
		if dy, ok := moveData["dy"].(float64); ok {
			newY := player.Y + dy
			if newY >= 0 && newY <= game.WorldHeight {
				player.Y = newY
			}
		}
	}
}

func checkCollisions(playerID string) {
	game.mu.Lock()
	defer game.mu.Unlock()

	player, ok := game.Players[playerID]
	if !ok {
		return
	}

	for collectibleID, collectible := range game.Collectibles {
		if distance(player.X, player.Y, collectible.X, collectible.Y) < (player.Radius + collectible.Radius) {
			player.Score += collectible.Points
			delete(game.Collectibles, collectibleID)
			spawnNewCollectible()
		}
	}
}

func distance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
}

func spawnNewCollectible() {
	game.CollectibleID++
	id := fmt.Sprintf("collectible_%d", game.CollectibleID)
	collectible := &Collectible{
		ID:     id,
		X:      rand.Float64() * game.WorldWidth,
		Y:      rand.Float64() * game.WorldHeight,
		Radius: 10,
		Color:  "#FFD700",
		Points: 10,
	}
	game.Collectibles[id] = collectible
}

func sendGameState(conn *websocket.Conn) {
	game.mu.RLock()
	gameState := GameState{
		Players:      game.Players,
		Collectibles: game.Collectibles,
		WorldWidth:   game.WorldWidth,
		WorldHeight:  game.WorldHeight,
	}
	game.mu.RUnlock()

	message := Message{
		Type:    "gameState",
		Payload: gameState,
	}

	if err := conn.WriteJSON(message); err != nil {
		log.Println("Failed to send game state:", err)
	}
}

func generatePlayerID() string {
	game.mu.Lock()
	defer game.mu.Unlock()
	return fmt.Sprintf("player_%d", len(game.Players)+1)
}

func spawnCollectiblesRoutine() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		game.mu.Lock()
		if len(game.Collectibles) < 5 {
			spawnNewCollectible()
		}
		game.mu.Unlock()
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	
	for i := 0; i < 5; i++ {
		spawnNewCollectible()
	}
	
	go spawnCollectiblesRoutine()

	http.HandleFunc("/ws", handleWebSocket)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
