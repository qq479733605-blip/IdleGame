package service

import (
	"errors"
	"math/rand"
	"time"

	"idlemmoserver/internal/domain"
	"idlemmoserver/internal/logx"
)

// SequenceResult represents the outcome returned from the sequence actor.
type SequenceResult struct {
	Gains        int64
	Rare         []string
	Items        []domain.Item
	SeqID        string
	Level        int
	CurExp       int64
	Leveled      bool
	SubProjectID string
}

// SequenceService encapsulates gameplay logic related to sequences.
type SequenceService struct {
	model *domain.PlayerModel
}

func NewSequenceService(model *domain.PlayerModel) *SequenceService {
	return &SequenceService{model: model}
}

func (s *SequenceService) CalculateOfflineRewards() (int64, map[string]int64, time.Duration) {
	if s.model.OfflineStart.IsZero() {
		return 0, map[string]int64{}, 0
	}
	duration := time.Since(s.model.OfflineStart)
	if duration <= 0 || duration >= s.model.OfflineLimit {
		return 0, map[string]int64{}, duration
	}

	gains := int64(0)
	items := make(map[string]int64)
	seconds := duration.Seconds()

	for seqID, level := range s.model.SeqLevels {
		if level <= 0 {
			continue
		}
		cfg, exists := domain.GetSequenceConfig(seqID)
		if !exists || cfg == nil {
			continue
		}
		interval := cfg.TickInterval
		if interval <= 0 {
			interval = 1
		}
		ticks := int64(seconds / float64(interval))
		if ticks <= 0 {
			continue
		}

		gain := cfg.BaseGain + int64(float64(level)*cfg.GrowthFactor)
		gains += gain * ticks

		for _, drop := range cfg.Drops {
			if drop.DropChance <= 0 {
				continue
			}
			expected := float64(ticks) * drop.DropChance
			guaranteed := int64(expected)
			remainder := expected - float64(guaranteed)
			count := guaranteed
			if rand.Float64() < remainder {
				count++
			}
			if count > 0 {
				items[drop.ID] += count
			}
		}
	}

	return gains, items, duration
}

func (s *SequenceService) ApplyOfflineRewards(gains int64, offlineItems map[string]int64) {
	if gains > 0 {
		s.model.Exp += gains
	}
	for itemID, count := range offlineItems {
		if err := s.model.Inventory.AddItem(domain.Item{ID: itemID, Name: itemID}, count); err != nil {
			logx.Warn("offline reward add item failed", "playerID", s.model.PlayerID, "itemID", itemID, "count", count, "err", err)
		}
	}
}

func (s *SequenceService) BuildReconnectedPayload() map[string]any {
	return map[string]any{
		"type":               "S_Reconnected",
		"msg":                "重连成功",
		"seq_id":             s.model.CurrentSeqID,
		"seq_level":          s.currentSeqLevel(),
		"exp":                s.model.Exp,
		"bag":                s.model.Inventory.List(),
		"is_running":         s.model.CurrentSeqID != "",
		"seq_levels":         s.model.SeqLevels,
		"equipment":          s.model.Equipment.Export(),
		"equipment_bonus":    s.model.Equipment.TotalBonus(),
		"active_sub_project": s.model.ActiveSubProject,
	}
}

func (s *SequenceService) currentSeqLevel() int {
	if s.model.CurrentSeqID == "" {
		return 0
	}
	return s.model.SeqLevels[s.model.CurrentSeqID]
}

func (s *SequenceService) PrepareStartSequence(seqID, subProjectID string) (*domain.SequenceConfig, *domain.SequenceSubProject, int, error) {
	cfg, exists := domain.GetSequenceConfig(seqID)
	if !exists {
		return nil, nil, 0, errors.New("sequence not found")
	}

	level := s.model.SeqLevels[seqID]
	var subProject *domain.SequenceSubProject
	if subProjectID != "" {
		sp, ok := cfg.GetSubProject(subProjectID)
		if !ok {
			return nil, nil, 0, errors.New("sub project not found")
		}
		if level < sp.UnlockLevel {
			return nil, nil, 0, errors.New("子项目未解锁")
		}
		subProject = sp
	}

	return cfg, subProject, level, nil
}

func (s *SequenceService) OnSequenceStarted(seqID, subProjectID string) {
	s.model.CurrentSeqID = seqID
	s.model.ActiveSubProject = subProjectID
}

func (s *SequenceService) OnSequenceStopped() {
	s.model.CurrentSeqID = ""
	s.model.ActiveSubProject = ""
}

func (s *SequenceService) ApplySequenceResult(result SequenceResult) map[string]any {
	for _, it := range result.Items {
		if err := s.model.Inventory.AddItem(it, 1); err != nil {
			logx.Error("Failed to add item", "itemID", it.ID, "error", err)
		}
	}
	if result.SeqID != "" {
		if s.model.SeqLevels == nil {
			s.model.SeqLevels = make(map[string]int)
		}
		s.model.SeqLevels[result.SeqID] = result.Level
	}
	s.model.ActiveSubProject = result.SubProjectID
	s.model.Exp += result.Gains

	payload := map[string]any{
		"type":            "S_SeqResult",
		"gains":           result.Gains,
		"rare":            result.Rare,
		"bag":             s.model.Inventory.List(),
		"seq_id":          result.SeqID,
		"level":           result.Level,
		"cur_exp":         result.CurExp,
		"leveled":         result.Leveled,
		"items":           result.Items,
		"sub_project_id":  result.SubProjectID,
		"equipment_bonus": s.model.Equipment.TotalBonus(),
	}
	return payload
}
