package math

import (
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestMathControllerScoringAndBandit(t *testing.T) {
	mc := NewMathController(models.MathL3)
	now := time.Now().UTC()
	rec := models.MemoryItem{
		ID:             "m1",
		Content:        "math test",
		Layer:          models.LayerSemantic,
		Importance:     0.9,
		Strength:       1.0,
		UsageCount:     4,
		CreatedAt:      now.Add(-2 * time.Hour),
		LastAccessedAt: now.Add(-1 * time.Hour),
	}
	score, breakdown := mc.ScoreMemory(rec, 0.85, now)
	if score <= 0 || score > 1.0 || breakdown["similarity"] <= 0 {
		t.Fatalf("unexpected ScoreMemory: score=%f, breakdown=%+v", score, breakdown)
	}

	weights := mc.UpdateReward("hyper_37", 0.95)
	if weights.ActiveArm != "hyper_37" {
		t.Fatalf("expected active arm hyper_37, got %s", weights.ActiveArm)
	}

	decayed := ComputeDecayedStrength(1.0, 2, 24.0, 0.05)
	if decayed <= 0 || decayed >= 1.0 {
		t.Fatalf("expected decayed strength in (0,1), got %f", decayed)
	}

	q8 := QuantizeInt8([]float32{-1.0, 0.0, 1.0})
	if len(q8) != 3 || q8[0] != -127 || q8[2] != 127 {
		t.Fatalf("unexpected QuantizeInt8: %+v", q8)
	}
	qb := QuantizeBinary([]float32{1.0, -1.0, 1.0, -1.0})
	if len(qb) != 1 {
		t.Fatalf("unexpected QuantizeBinary length: %d", len(qb))
	}
}
