package player

type StopSequenceCommand struct{}

func NewStopSequenceCommand(_ []byte) Command {
	return &StopSequenceCommand{}
}

func (c *StopSequenceCommand) Execute(ctx *CommandContext) error {
	if ctx.Player.currentSeq != nil {
		ctx.Services.StopSequence(ctx.ActorCtx, ctx.Player.currentSeq)
		ctx.Player.currentSeq = nil
		ctx.Player.currentSeqID = ""
		ctx.Player.state.ActiveSubProject = ""
		ctx.Player.sendToClient(map[string]any{
			"type":               "S_SeqEnded",
			"is_running":         false,
			"seq_id":             "",
			"seq_level":          0,
			"active_sub_project": "",
		})
	}
	return nil
}
