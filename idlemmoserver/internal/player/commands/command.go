package player

import "github.com/asynkron/protoactor-go/actor"

type Command interface {
	Execute(ctx *CommandContext) error
}

type CommandContext struct {
	ActorCtx   actor.Context
	Domain     *Domain
	Services   *Services
	Player     *PlayerActor
	RawPayload []byte
}

type CommandFactory func(raw []byte) Command
