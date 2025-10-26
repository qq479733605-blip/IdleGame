package common

import "github.com/asynkron/protoactor-go/actor"

type PlayerRepository interface {
	SavePlayer(snapshot PlayerSnapshot) error
	LoadPlayer(playerID string) (*PlayerSnapshot, error)
}

type SequenceManager interface {
	StartSequence(req SequenceStartRequest) (*actor.PID, error)
	StopSequence(pid *actor.PID)
}

type UserRepository interface {
	SaveUser(user *UserData) error
	GetUser(username string) (*UserData, error)
	GetUserByPlayerID(playerID string) (*UserData, error)
	UpdateLastLogin(username string) error
	UserExists(username string) bool
}

type SequenceStartRequest struct {
	PlayerID     string
	SeqID        string
	Level        int
	SubProjectID string
	Parent       *actor.PID
	Scheduler    *actor.PID
	Bonus        EquipmentBonus
}
