package common

import "github.com/asynkron/protoactor-go/actor"

type MsgFromWS struct {
	Conn interface{}
	Data []byte
}

type MsgWSClosed struct{ Conn interface{} }

type MsgClientPayload struct {
	Conn interface{}
	Raw  []byte
}

type MsgConnClosed struct{ Conn interface{} }

type MsgPlayerOffline struct{}

type MsgPlayerReconnect struct{ Conn interface{} }

type MsgCheckExpire struct{}

type MsgSavePlayer struct {
	Snapshot PlayerSnapshot
}

type MsgLoadPlayer struct {
	PlayerID string
	ReplyTo  *actor.PID
}

type MsgLoadResult struct {
	Snapshot *PlayerSnapshot
	Err      error
}

type MsgRegisterPlayer struct {
	PlayerID string
	PID      *actor.PID
}

type MsgUnregisterPlayer struct{ PlayerID string }

type MsgEnsurePlayer struct {
	PlayerID string
	ReplyTo  *actor.PID
}

type MsgPlayerReady struct {
	PlayerPID *actor.PID
}

type MsgRegisterUser struct {
	Username string
	Password string
	ReplyTo  *actor.PID
}

type MsgRegisterUserResult struct {
	Success  bool
	Message  string
	PlayerID string
}

type MsgAuthenticateUser struct {
	Username string
	Password string
	ReplyTo  *actor.PID
}

type MsgAuthenticateUserResult struct {
	Success  bool
	Message  string
	PlayerID string
}

type MsgGetUserByPlayerID struct {
	PlayerID string
	ReplyTo  *actor.PID
}

type MsgGetUserByPlayerIDResult struct {
	User   *UserData
	Exists bool
}

type MsgSequenceTick struct{}

type MsgSequenceStop struct{}

type MsgSequenceResult struct {
	PlayerID     string
	SeqID        string
	Items        []ItemDrop
	Rare         []string
	Gains        int64
	Level        int
	CurExp       int64
	Leveled      bool
	SubProjectID string
}

type MsgUpdateEquipmentBonus struct {
	Bonus EquipmentBonus
}
