# Neal: Real-Time Multiplayer Game Server

This project is a WebSocket-based real-time server implemented in Go, designed to manage a multiplayer game with power-ups, collectibles, and player interactions in a shared game world.

## Features

- **Real-Time Player Movement**: Allows players to move within a predefined game area.
- **Collectibles**: Players can collect items for points.
- **Power-Ups**: Players can get temporary power-ups with various effects like increased speed and point multipliers.
- **WebSocket Communication**: Game state is synchronized between server and clients in real-time.
- **Concurrent Handling**: Game state is managed safely with concurrent access using Go's `sync` package.

## Installation

1. **Clone the Repository**:
   ```bash
   git clone <repository-url>
   cd <repository-directory>
   ```

2. **Install Dependencies**:
   Ensure Go is installed on your system, and then install required packages:
   ```bash
   go get github.com/gorilla/websocket
   ```

3. **Run the Server**:
   Start the server on port 8080:
   ```bash
   go run main.go
   ```
   The server will listen for WebSocket connections at `ws://localhost:8080/ws`.

4. Either develop your own game or you can start a web server and host the test_game to play a basic example of the game.

## Game Mechanics

- **Players**: Each player has attributes like position, score, radius, and active power-ups.
- **Collectibles**: Items that players can collect to gain points, which respawn at regular intervals.
- **Power-Ups**: Power-ups provide temporary effects such as speed boosts, size increases, and points multipliers.

## API

### WebSocket Endpoint

- **Endpoint**: `ws://localhost:8080/ws`
- **Messages**: Clients can send and receive JSON messages to update game state and handle player interactions.

#### Message Types

1. **Move**: Update the player's position based on movement data.
   ```json
   {
       "type": "move",
       "payload": { "dx": <float>, "dy": <float> }
   }
   ```
2. **Game State**: Sent by the server to broadcast the current game state to all connected clients.
   ```json
   {
       "type": "gameState",
       "payload": { "players": {}, "collectibles": {}, "powerUps": {} }
   }
   ```

## Running the Server

1. **Collectibles and Power-Ups Spawning**:
   - Collectibles respawn every 5 seconds.
   - Power-ups spawn every 15 seconds if there are fewer than three on the map.

2. **Player Power-Ups**: Power-up effects last for 10 seconds and revert after expiration.