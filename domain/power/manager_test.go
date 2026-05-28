package power

import (
	"testing"
	"time"
)

func TestPulseClassifier(t *testing.T) {
	rebootMin := 200
	rebootMax := 600

	classifier := NewPulseClassifier(rebootMin, rebootMax)

	// Test 1: Short pulse (100ms) -> ActionNone
	action := classifier.Classify(100 * time.Millisecond)
	if action != ActionNone {
		t.Errorf("Expected ActionNone for 100ms, got %s", action)
	}

	// Test 2: Mid pulse (400ms) -> ActionReboot
	action = classifier.Classify(400 * time.Millisecond)
	if action != ActionReboot {
		t.Errorf("Expected ActionReboot for 400ms, got %s", action)
	}

	// Test 3: Long pulse (1000ms) -> ActionPowerOff
	action = classifier.Classify(1000 * time.Millisecond)
	if action != ActionPowerOff {
		t.Errorf("Expected ActionPowerOff for 1000ms, got %s", action)
	}
}
