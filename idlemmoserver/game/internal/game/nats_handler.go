package game

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"idlemmoserver/common"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/nats-io/nats.go"
)

type Server struct {
	nc      *nats.Conn
	system  *actor.ActorSystem
	root    *actor.RootContext
	players sync.Map
}

func NewServer(nc *nats.Conn) *Server {
	system := actor.NewActorSystem()
	return &Server{
		nc:     nc,
		system: system,
		root:   system.Root,
	}
}

func (s *Server) Start(ctx context.Context) error {
	if _, err := s.nc.Subscribe(common.SubjectGateEnsurePlayer, s.handleEnsurePlayer); err != nil {
		return err
	}
	if _, err := s.nc.Subscribe(common.SubjectGateClient, s.handleClientMessage); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		s.nc.Drain()
	}()

	return nil
}

func (s *Server) handleEnsurePlayer(msg *nats.Msg) {
	var req common.GatewayEnsurePlayerRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return
	}
	pid := s.ensurePlayer(req.PlayerID)
	resp := common.GatewayEnsurePlayerResponse{PlayerID: req.PlayerID, Success: pid != nil}
	if pid == nil {
		resp.Success = false
		resp.Error = "failed to spawn player actor"
	}
	if err := msg.Respond(common.MustMarshal(resp)); err != nil {
		log.Printf("respond ensure player: %v", err)
	}
}

func (s *Server) handleClientMessage(msg *nats.Msg) {
	var payload common.GateClientMessage
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		return
	}
	pid := s.ensurePlayer(payload.PlayerID)
	if pid == nil {
		return
	}
	s.root.Send(pid, &clientCommand{Raw: payload.Message})
}

func (s *Server) ensurePlayer(playerID string) *actor.PID {
	if value, ok := s.players.Load(playerID); ok {
		return value.(*actor.PID)
	}

	props := actor.PropsFromProducer(func() actor.Actor {
		return NewPlayerActor(playerID, s.nc)
	})
	pid := s.root.Spawn(props)
	actual, loaded := s.players.LoadOrStore(playerID, pid)
	if loaded {
		s.root.Stop(pid)
		return actual.(*actor.PID)
	}
	return pid
}
