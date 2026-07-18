package gamelogic

import (
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/Khaz713/Tag_and_Seek/internal/routing"
)

const (
	HiderCooldown          = 400 * time.Millisecond //how often can a hider move
	SeekerCooldown         = 200 * time.Millisecond // how often can a seeker move
	HiderVision    float64 = 8.0
	SeekerVision   float64 = 4.0
	SpecatorVision float64 = 999.0
)

type ServerPlayer struct {
	ID           string
	X            int
	Y            int
	IsSeeker     bool
	IsPlaying    bool
	LastMoveTime time.Time
}

type GameRoom struct {
	ID         string
	Map        []string
	Players    map[string]*ServerPlayer
	State      routing.RoomState
	GameWinner string
	Size       int
}

func (room *GameRoom) findSpawnPoint() (int, int) {
	maxY := len(room.Map)
	maxX := len(room.Map[0])

	for {
		x := rand.Intn(maxX)
		y := rand.Intn(maxY)

		if room.Map[y][x] == ' ' {
			return x, y
		}
	}
}

func (room *GameRoom) seekerSpawnPoint() (int, int) {
	maxY := len(room.Map)
	maxX := len(room.Map[0])
	return maxX / 2, maxY / 2
}

func (room *GameRoom) AddPlayer(playerID string) {
	isFirstPlayer := len(room.Players) == 0 //if the player is the first that joins the room he will be seeking first

	x, y := room.seekerSpawnPoint()

	if !isFirstPlayer {
		x, y = room.findSpawnPoint()
	}

	room.Players[playerID] = &ServerPlayer{
		ID:           playerID,
		X:            x,
		Y:            y,
		IsSeeker:     isFirstPlayer,
		IsPlaying:    true,
		LastMoveTime: time.Now(),
	}

	if len(room.Players) == room.Size && room.State == routing.StateLobby {
		room.State = routing.StatePlaying
	}
}

func (room *GameRoom) MovePlayer(playerID string, dx, dy int) error {
	if room.State != routing.StatePlaying { //can only move if the game is active
		return errors.New("cannot move: game not in progress")
	}
	player, exists := room.Players[playerID]
	if !exists {
		return errors.New("cannot move: player not found")
	}
	now := time.Now() //set movement cooldown based on the role
	cooldown := HiderCooldown
	if player.IsSeeker {
		cooldown = SeekerCooldown
	}

	if now.Sub(player.LastMoveTime) < cooldown {
		return errors.New("cannot move: movement on cooldown")
	}

	newX := player.X + dx //calculates new position
	newY := player.Y + dy

	//check if new position is in the bounds of the map
	if newX < 0 || newY < 0 || newX >= len(room.Map[0]) || newY >= len(room.Map) {
		return errors.New("cannot move: out of bounds")
	}

	//check if new position is not colliding with walls or other hiders(no need to check for seekers, as the condition for being caught is to be next to the seeker)
	tile := room.Map[newY][newX]
	if tile == '-' || tile == '|' || tile == '*' || tile == 'S' {
		return errors.New("cannot move: wall/players in the way")
	}

	//update position and cooldown timer
	player.X = newX
	player.Y = newY
	player.LastMoveTime = now

	room.checkIfTagged()
	return nil
}

func (room *GameRoom) canSee(p1X, p1Y, p2X, p2Y int, vision float64) bool { //checks if the player can see another player
	dx := float64(p1X - p2X)
	dy := float64(p1Y - p2Y)
	distance := math.Sqrt(dx*dx + dy*dy)
	return distance <= vision
}

func (room *GameRoom) checkIfTagged() { //checks if any of the hiders has been tagged by the seeker
	var seeker *ServerPlayer
	var hiders []*ServerPlayer

	for _, p := range room.Players {
		if p.IsSeeker {
			seeker = p
		} else {
			if p.IsPlaying {
				hiders = append(hiders, p)
			}
		}
	}

	for _, hider := range hiders {
		dx := math.Abs(float64(seeker.X - hider.X))
		dy := math.Abs(float64(seeker.Y - hider.Y))

		if dx <= 1 && dy <= 1 {
			hider.IsPlaying = false
		}
	}
	if len(hiders) == 0 {
		room.State = routing.StateEnded
		return
	}
}

func (room *GameRoom) GetStateForPlayer(playerID string) routing.GameStateUpdate {
	targetPlayer := room.Players[playerID]
	visiblePlayers := make(map[string]routing.PlayerState)

	visionRadius := HiderVision
	if targetPlayer.IsSeeker {
		visionRadius = SeekerVision
	}
	if !targetPlayer.IsPlaying {
		visionRadius = SpecatorVision
	}

	for id, p := range room.Players {
		if id == playerID && targetPlayer.IsPlaying {
			visiblePlayers[id] = routing.PlayerState{
				ID:        targetPlayer.ID,
				X:         targetPlayer.X,
				Y:         targetPlayer.Y,
				IsSeeker:  targetPlayer.IsSeeker,
				IsPlaying: targetPlayer.IsPlaying,
			}
			continue
		}
		if room.canSee(targetPlayer.X, targetPlayer.Y, p.X, p.Y, visionRadius) {
			visiblePlayers[id] = routing.PlayerState{
				ID:        p.ID,
				X:         p.X,
				Y:         p.Y,
				IsSeeker:  p.IsSeeker,
				IsPlaying: p.IsPlaying,
			}
		}
	}

	return routing.GameStateUpdate{
		RoomID:  room.ID,
		State:   room.State,
		Players: visiblePlayers,
	}
}
