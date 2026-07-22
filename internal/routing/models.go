package routing

import "time"

type PlayerCommand struct { //the command that player sends to the server
	PlayerID string
	RoomID   string
	Command  string //can be 2 options, MOVE_x where x is the direction of movement, or ROOM_y where y can be JOIN or LEAVE
}

type PlayerState struct {
	ID         string
	X          int
	Y          int
	IsSeeker   bool
	IsPlaying  bool
	BeenSeeker bool
}

type RoomState string

const (
	StateLobby    RoomState = "LOBBY"
	StatePlaying  RoomState = "PLAYING"
	StateRoundEnd RoomState = "ROUND_END"
	StateEnded    RoomState = "ENDED"
)

type GameStateUpdate struct {
	RoomID     string
	MapIndex   int
	State      RoomState
	Players    map[string]PlayerState
	GameWinner string
}

type PlayerEnd struct {
	PlayerID string
	Duration int
	Ranking  int
}
type GameResult struct {
	GameID   string
	WinnerID string
	Players  []PlayerEnd
	Duration int
	MapIndex int
	PlayedAt time.Time
}

type RegisterRequest struct {
	Username string
	Password string
}

type RegisterResponse struct {
	UserID   string
	Token    string
	Username string
}

type LogoutRequest struct {
	Token string
}

type TokenIdentification struct {
	Token string
}

type GamePlayers struct {
	UserID        string
	Username      string
	HiddenSeconds int
	Ranking       int
}
type GameHistory struct {
	GameID          string
	MapIndex        int
	WinnerID        string
	DurationSeconds int
	PlayedAt        time.Time
	Players         []GamePlayers
}

type GameHistoryResponse struct {
	Games []GameHistory
}

type CreateRoomRequest struct {
	Token string
	Name  string
	Size  int
}

type CreateRoomResponse struct {
	RoomID string
}

type RoomInfo struct {
	ID         string
	Name       string
	Size       int
	PlayersNum int
}
type GetRoomsResponse struct {
	Rooms []RoomInfo
}

type JoinRoomRequest struct {
	Token    string
	RoomName string
}
