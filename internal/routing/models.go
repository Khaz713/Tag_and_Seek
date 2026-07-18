package routing

import "time"

type PlayerCommand struct { //the command that player sends to the server
	PlayerID string
	RoomID   string
	Command  string //can be 2 options, MOVE_x where x is the direction of movement, or ROOM_y where y can be JOIN or LEAVE
}

type PlayerState struct {
	ID        string
	X         int
	Y         int
	IsSeeker  bool
	IsPlaying bool
}

type RoomState string

const (
	StateLobby   RoomState = "LOBBY"
	StatePlaying RoomState = "PLAYING"
	StateEnded   RoomState = "ENDED"
)

type GameStateUpdate struct {
	RoomID     string
	MapIndex   int
	State      RoomState
	Players    map[string]PlayerState
	GameWinner string
}

type MatchResult struct {
	RoomID   string
	WinnerID string
	LooserID []string
	Duration int
	EndedAt  time.Time
}
