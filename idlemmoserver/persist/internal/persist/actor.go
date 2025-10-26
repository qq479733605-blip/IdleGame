package persist

import (
	"context"
	"encoding/json"
	"log"

	"idlemmoserver/common"

	"github.com/nats-io/nats.go"
)

type Service struct {
	repo Repository
	nc   *nats.Conn
}

func NewService(nc *nats.Conn, repo Repository) *Service {
	return &Service{repo: repo, nc: nc}
}

func (s *Service) Start(ctx context.Context) error {
	if _, err := s.nc.Subscribe(common.SubjectPersistSave, s.handleSave); err != nil {
		return err
	}
	if _, err := s.nc.Subscribe(common.SubjectPersistLoad, s.handleLoad); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		s.nc.Drain()
	}()

	return nil
}

func (s *Service) handleSave(msg *nats.Msg) {
	var req common.PersistSaveRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return
	}
	if req.Snapshot.PlayerID == "" {
		return
	}
	if err := s.repo.Save(req.Snapshot); err != nil {
		log.Printf("persist save %s: %v", req.Snapshot.PlayerID, err)
	}
}

func (s *Service) handleLoad(msg *nats.Msg) {
	var req common.PersistLoadRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return
	}
	snapshot, ok, err := s.repo.Load(req.PlayerID)
	if err != nil {
		log.Printf("persist load %s: %v", req.PlayerID, err)
		return
	}
	resp := common.PersistLoadResponse{Snapshot: snapshot, Found: ok}
	if err := msg.Respond(common.MustMarshal(resp)); err != nil {
		log.Printf("persist respond %s: %v", req.PlayerID, err)
	}
}
