package actors

import (
	"time"

	"idlemmoserver/internal/controller"
	"idlemmoserver/internal/domain"
	"idlemmoserver/internal/logx"
	"idlemmoserver/internal/service"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

type PlayerActor struct {
	model        *domain.PlayerModel
	controller   *controller.PlayerController
	persistPID   *actor.PID
	schedulerPID *actor.PID
	currentSeq   *actor.PID
}

func NewPlayerActor(playerID string, root *actor.RootContext, persistPID *actor.PID, schedulerPID *actor.PID) actor.Actor {
	_ = root
	model := domain.NewPlayerModel(playerID)
	actor := &PlayerActor{
		model:        model,
		persistPID:   persistPID,
		schedulerPID: schedulerPID,
	}
	actor.controller = controller.NewPlayerController(model, actor)
	return actor
}

type MsgAttachConn struct {
	Conn         *websocket.Conn
	RequestState bool
}

type MsgDetachConn struct{}

type SeqResult struct {
	Gains        int64
	Rare         []string
	Items        []domain.Item
	SeqID        string
	Level        int
	CurExp       int64
	Leveled      bool
	SubProjectID string
}

func (p *PlayerActor) Receive(ctx actor.Context) {
	switch m := ctx.Message().(type) {
	case *actor.Started:
		ctx.Send(p.persistPID, &MsgRegisterPlayer{PlayerID: p.model.PlayerID, PID: ctx.Self()})
		ctx.Send(p.persistPID, &MsgLoadPlayer{PlayerID: p.model.PlayerID, ReplyTo: ctx.Self()})
		p.model.LastActive = time.Now()

	case *MsgAttachConn:
		p.controller.AttachConn(ctx, m.Conn, m.RequestState)

	case *MsgDetachConn:
		p.controller.DetachConn()

	case *MsgLoadResult:
		p.handleLoadResult(m)

	case *MsgClientPayload:
		p.controller.HandleClientPayload(ctx, m.Conn, m.Raw)

	case *SeqResult:
		p.handleSeqResult(ctx, m)

	case *MsgPlayerOffline:
		p.controller.HandlePlayerOffline()

	case *MsgPlayerReconnect:
		p.controller.HandleReconnect(m.Conn)

	case *MsgCheckExpire:
		if !p.model.IsOnline && !p.model.OfflineStart.IsZero() && time.Since(p.model.OfflineStart) > p.model.OfflineLimit {
			ctx.Send(p.persistPID, &MsgSavePlayer{
				PlayerID:          p.model.PlayerID,
				SeqLevels:         p.model.SeqLevels,
				Inventory:         p.model.Inventory,
				Exp:               p.model.Exp,
				Equipment:         p.model.Equipment.ExportState(),
				OfflineLimitHours: int64(p.model.OfflineLimit / time.Hour),
			})
			ctx.Send(p.persistPID, &MsgUnregisterPlayer{PlayerID: p.model.PlayerID})
			logx.Warn("player session expired", "player", p.model.PlayerID)
			ctx.Stop(ctx.Self())
		}

	case *MsgConnClosed:
		p.controller.HandleConnClosed(m.Conn)

	case *actor.Terminated:
		if p.currentSeq != nil && m.Who.Equal(p.currentSeq) {
			p.currentSeq = nil
			p.controller.OnSequenceTerminated()
		}
	}
}

func (p *PlayerActor) handleLoadResult(m *MsgLoadResult) {
	if m.Err != nil || m.Data == nil {
		p.model.SeqLevels = make(map[string]int)
		for seqID := range domain.Sequences {
			p.model.SeqLevels[seqID] = 1
		}
		p.controller.SendNewPlayer()
		return
	}

	p.model.SeqLevels = m.Data.SeqLevels
	if p.model.SeqLevels == nil {
		p.model.SeqLevels = make(map[string]int)
	}
	for id, cnt := range m.Data.Inventory {
		_ = p.model.Inventory.AddItem(domain.Item{ID: id, Name: id}, cnt)
	}
	p.model.Exp = m.Data.Exp
	if m.Data.Equipment != nil {
		p.model.Equipment.ImportState(m.Data.Equipment)
	}
	if m.Data.OfflineLimitHours > 0 {
		p.model.OfflineLimit = time.Duration(m.Data.OfflineLimitHours) * time.Hour
	}

	p.controller.SendLoadOK(m.Data.OfflineLimitHours)
}

func (p *PlayerActor) handleSeqResult(ctx actor.Context, m *SeqResult) {
	logx.Info("Player received SeqResult", "playerID", p.model.PlayerID, "seqID", m.SeqID, "gains", m.Gains, "items", len(m.Items), "isOnline", p.model.IsOnline)

	result := service.SequenceResult{
		Gains:        m.Gains,
		Rare:         m.Rare,
		Items:        m.Items,
		SeqID:        m.SeqID,
		Level:        m.Level,
		CurExp:       m.CurExp,
		Leveled:      m.Leveled,
		SubProjectID: m.SubProjectID,
	}
	p.controller.HandleSequenceResult(ctx, result)
}

// PlayerRuntime interface implementation

func (p *PlayerActor) CurrentSequence() *actor.PID {
	return p.currentSeq
}

func (p *PlayerActor) StartSequence(ctx actor.Context, seqID string, level int, subProject *domain.SequenceSubProject, bonus domain.EquipmentBonus) (*actor.PID, error) {
	pid := ctx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return NewSequenceActor(p.model.PlayerID, seqID, level, ctx.Self(), p.schedulerPID, subProject, bonus)
	}))
	ctx.Watch(pid)
	p.currentSeq = pid
	return pid, nil
}

func (p *PlayerActor) StopCurrentSequence(ctx actor.Context) bool {
	if p.currentSeq != nil {
		ctx.Stop(p.currentSeq)
		p.currentSeq = nil
		return true
	}
	return false
}

func (p *PlayerActor) PushEquipmentBonus(ctx actor.Context) {
	if p.currentSeq != nil {
		ctx.Send(p.currentSeq, &MsgUpdateEquipmentBonus{Bonus: p.model.Equipment.TotalBonus()})
	}
}

func (p *PlayerActor) PersistPlayer(ctx actor.Context) {
	ctx.Send(p.persistPID, &MsgSavePlayer{
		PlayerID:          p.model.PlayerID,
		SeqLevels:         p.model.SeqLevels,
		Inventory:         p.model.Inventory,
		Exp:               p.model.Exp,
		Equipment:         p.model.Equipment.ExportState(),
		OfflineLimitHours: int64(p.model.OfflineLimit / time.Hour),
	})
}
