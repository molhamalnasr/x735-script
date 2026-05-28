package fan

import "testing"

func TestHysteresisController(t *testing.T) {
	thresholds := []int{25, 40, 50, 60, 70, 75}
	dutyCycles := []int{40, 45, 50, 70, 80, 100}
	hysteresis := 2.0

	ctrl := NewHysteresisController(thresholds, dutyCycles, hysteresis)

	// Test 1: Initial run at 30°C should match 40% (25 <= 30 < 40)
	duty := ctrl.CalculateDutyCycle(30.0)
	if duty != 40 {
		t.Errorf("Expected initial duty 40 at 30°C, got %d", duty)
	}

	// Test 2: Temperature rises to 52°C. Speed should increase to 50% immediately
	duty = ctrl.CalculateDutyCycle(52.0)
	if duty != 50 {
		t.Errorf("Expected duty 50 at 52°C, got %d", duty)
	}

	// Test 3: Temperature falls to 49.5°C. With hysteresis of 2.0°C, the cooldown
	// threshold to drop from 50% is 50.0 - 2.0 = 48.0°C.
	// Since 49.5°C is >= 48.0°C, it should STAY at 50%.
	duty = ctrl.CalculateDutyCycle(49.5)
	if duty != 50 {
		t.Errorf("Expected duty to stay at 50 at 49.5°C (cooldown threshold 48°C), got %d", duty)
	}

	// Test 4: Temperature falls further to 47.5°C (< 48.0°C).
	// It should now drop to the standard speed of 45% (since 40 <= 47.5 < 50).
	duty = ctrl.CalculateDutyCycle(47.5)
	if duty != 45 {
		t.Errorf("Expected duty to drop to 45 at 47.5°C, got %d", duty)
	}
}
