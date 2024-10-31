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

type PowerUpType string

const (
	SpeedBoost       PowerUpType = "speedBoost"
	SizeIncrease     PowerUpType = "sizeIncrease"
	PointsMultiplier PowerUpType = "pointsMultiplier"
)

type PowerUp struct {
	ID       string     `json:"id"`
	Type     PowerUpType `json:"type"`
	X        float64    `json:"x"`
	Y        float64    `json:"y"`
	Radius   float64    `json:"radius"`
	Color    string     `json:"color"`
	Duration float64    `json:"duration"`
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
	ActivePowerUps    map[PowerUpType]time.Time `json:"activePowerUps"`
	SpeedMultiplier   float64                   `json:"speedMultiplier"`
	PointsMultiplier  int                       `json:"pointsMultiplier"`
	OriginalRadius    float64                   `json:"originalRadius"`
}

type Game struct {
	Players       map[string]*Player
	Collectibles  map[string]*Collectible
	PowerUps      map[string]*PowerUp
	mu            sync.RWMutex
	WorldWidth    float64
	WorldHeight   float64
	CollectibleID int
	PowerUpID     int
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
	PowerUps     map[string]*PowerUp    `json:"powerUps"`
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
		PowerUps:     make(map[string]*PowerUp),
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
	return &Player{
		ID:              id,
		X:               rand.Float64() * game.WorldWidth,
		Y:               rand.Float64() * game.WorldHeight,
		Radius:          20,
		Color:           colors[rand.Intn(len(colors))],
		Score:           0,
		ActivePowerUps:  make(map[PowerUpType]time.Time),
		SpeedMultiplier: 1.0,
		PointsMultiplier: 1,
		OriginalRadius:   20,
	}
}

func spawnNewPowerUp() {
	game.PowerUpID++
	id := fmt.Sprintf("powerup_%d", game.PowerUpID)
	
	powerUpTypes := []PowerUpType{SpeedBoost, SizeIncrease, PointsMultiplier}
	powerUpType := powerUpTypes[rand.Intn(len(powerUpTypes))]
	
	var color string
	switch powerUpType {
	case SpeedBoost:
		color = "#FF0000"
	case SizeIncrease:
		color = "#00FF00"
	case PointsMultiplier:
		color = "#0000FF"
	}
	
	game.PowerUps[id] = &PowerUp{
		ID:       id,
		Type:     powerUpType,
		X:        rand.Float64() * game.WorldWidth,
		Y:        rand.Float64() * game.WorldHeight,
		Radius:   15,
		Color:    color,
		Duration: 10,
	}
}

func spawnNewCollectible() {
	game.CollectibleID++
	id := fmt.Sprintf("collectible_%d", game.CollectibleID)
	game.Collectibles[id] = &Collectible{
		ID:     id,
		X:      rand.Float64() * game.WorldWidth,
		Y:      rand.Float64() * game.WorldHeight,
		Radius: 10,
		Color:  "#FFD700",
		Points: 10,
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
			dx *= player.SpeedMultiplier
			newX := player.X + dx
			if newX >= 0 && newX <= game.WorldWidth {
				player.X = newX
			}
		}
		if dy, ok := moveData["dy"].(float64); ok {
			dy *= player.SpeedMultiplier
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
			player.Score += collectible.Points * player.PointsMultiplier
			delete(game.Collectibles, collectibleID)
			spawnNewCollectible()
		}
	}

	for powerUpID, powerUp := range game.PowerUps {
		if distance(player.X, player.Y, powerUp.X, powerUp.Y) < (player.Radius + powerUp.Radius) {
			applyPowerUp(player, powerUp)
			delete(game.PowerUps, powerUpID)
		}
	}
}

func applyPowerUp(player *Player, powerUp *PowerUp) {
	expiryTime := time.Now().Add(time.Duration(powerUp.Duration) * time.Second)
	player.ActivePowerUps[powerUp.Type] = expiryTime

	switch powerUp.Type {
	case SpeedBoost:
		player.SpeedMultiplier = 2.0
	case SizeIncrease:
		player.Radius = player.OriginalRadius * 1.5
	case PointsMultiplier:
		player.PointsMultiplier = 2
	}

	go func(playerID string, powerUpType PowerUpType) {
		time.Sleep(time.Duration(powerUp.Duration) * time.Second)
		
		game.mu.Lock()
		defer game.mu.Unlock()
		
		if player, ok := game.Players[playerID]; ok {
			delete(player.ActivePowerUps, powerUpType)
			
			switch powerUpType {
			case SpeedBoost:
				player.SpeedMultiplier = 1.0
			case SizeIncrease:
				player.Radius = player.OriginalRadius
			case PointsMultiplier:
				player.PointsMultiplier = 1
			}
		}
	}(player.ID, powerUp.Type)
}

func distance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
}

func sendGameState(conn *websocket.Conn) {
	game.mu.RLock()
	gameState := GameState{
		Players:      game.Players,
		Collectibles: game.Collectibles,
		PowerUps:     game.PowerUps,
		WorldWidth:   game.WorldWidth,
		WorldHeight:  game.WorldHeight,
	}
	game.mu.RUnlock()

	if err := conn.WriteJSON(Message{Type: "gameState", Payload: gameState}); err != nil {
		log.Println("Failed to send game state:", err)
	}
}

func generatePlayerID() string {
	game.mu.Lock()
	defer game.mu.Unlock()
	return fmt.Sprintf("player_%d", len(game.Players)+1)
}

func spawnPowerUpsRoutine() {
	ticker := time.NewTicker(15 * time.Second)
	for range ticker.C {
		game.mu.Lock()
		if len(game.PowerUps) < 3 {
			spawnNewPowerUp()
		}
		game.mu.Unlock()
	}
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
	
	for i := 0; i < 2; i++ {
		spawnNewPowerUp()
	}
	
	go spawnCollectiblesRoutine()
	go spawnPowerUpsRoutine()

	http.HandleFunc("/ws", handleWebSocket)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
