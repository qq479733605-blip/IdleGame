package player

import (
	"encoding/json"

	"idlemmoserver/internal/logx"
	"idlemmoserver/internal/sequence"
)

type startSequenceRequest struct {
	SeqID        string `json:"seq_id"`
	Target       int64  `json:"target"`
	SubProjectID string `json:"sub_project_id"`
}

type StartSequenceCommand struct {
	req startSequenceRequest
}

func NewStartSequenceCommand(raw []byte) Command {
	var req startSequenceRequest
	_ = json.Unmarshal(raw, &req)
	return &StartSequenceCommand{req: req}
}

func (c *StartSequenceCommand) Execute(ctx *CommandContext) error {
	if ctx.Player.currentSeq != nil {
		ctx.Services.StopSequence(ctx.ActorCtx, ctx.Player.currentSeq)
		ctx.Player.currentSeq = nil
		ctx.Player.currentSeqID = ""
		ctx.Player.state.ActiveSubProject = ""
	}

	cfg, exists := sequence.GetConfig(c.req.SeqID)
	if !exists || cfg == nil {
		ctx.Player.sendError("sequence not found")
		return ErrSequenceNotFound
	}

	level := ctx.Domain.State().SeqLevels[c.req.SeqID]
	var sub *sequence.SubProject
	if c.req.SubProjectID != "" {
		sp, ok := cfg.GetSubProject(c.req.SubProjectID)
		if !ok {
			ctx.Player.sendError("sub project not found")
			return ErrSubProjectNotFound
		}
		if level < sp.UnlockLevel {
			ctx.Player.sendError("子项目未解锁")
			return ErrSubProjectLocked
		}
		sub = sp
	}

	bonus := ctx.Domain.State().Equipment.TotalBonus()
	params := sequence.Params{
		PlayerID:  ctx.Player.playerID,
		SeqID:     c.req.SeqID,
		Level:     level,
		Sub:       sub,
		Parent:    ctx.ActorCtx.Self(),
		Scheduler: ctx.Services.SchedulerPID,
		Bonus:     bonus,
	}
	pid := ctx.Services.SpawnSequence(ctx.ActorCtx, params)
	ctx.Player.currentSeq = pid
	ctx.Player.currentSeqID = c.req.SeqID
	if sub != nil {
		ctx.Player.state.ActiveSubProject = sub.ID
	} else {
		ctx.Player.state.ActiveSubProject = ""
	}

	interval := sequence.EffectiveInterval(cfg, sub)
	ctx.Player.sendToClient(map[string]any{
		"type":            "S_SeqStarted",
		"seq_id":          c.req.SeqID,
		"level":           level,
		"sub_project_id":  ctx.Player.state.ActiveSubProject,
		"tick_interval":   interval.Seconds(),
		"equipment_bonus": bonus,
	})
	logx.Info("sequence started", "player", ctx.Player.playerID, "seq", c.req.SeqID)
	return nil
}
