package fan

// HysteresisController handles the domain logic for calculating fan speeds based on
// CPU temperature, applying a hysteresis buffer to prevent rapid duty cycle oscillations.
type HysteresisController struct {
	thresholds  []float64
	dutyCycles  []int
	currentDuty int
	hysteresis  float64
}

// NewHysteresisController creates a new HysteresisController instance.
func NewHysteresisController(thresholds []int, dutyCycles []int, hysteresis float64) *HysteresisController {
	floatThresholds := make([]float64, len(thresholds))
	for i, v := range thresholds {
		floatThresholds[i] = float64(v)
	}

	return &HysteresisController{
		thresholds:  floatThresholds,
		dutyCycles:  dutyCycles,
		currentDuty: -1, // Uninitialized state
		hysteresis:  hysteresis,
	}
}

// CalculateDutyCycle calculates the target duty cycle for a given CPU temperature.
// It applies the hysteresis buffer when the temperature is decreasing.
func (h *HysteresisController) CalculateDutyCycle(temp float64) int {
	if len(h.thresholds) == 0 {
		return 0
	}

	// 1. Calculate the standard target duty cycle based on standard thresholds (rising temp logic)
	standardDuty := h.getStandardDuty(temp)

	// If uninitialized, set current to standard and return
	if h.currentDuty == -1 {
		h.currentDuty = standardDuty
		return h.currentDuty
	}

	// If the temp warrants a higher speed, increase speed immediately for safety
	if standardDuty > h.currentDuty {
		h.currentDuty = standardDuty
		return h.currentDuty
	}

	// If the temp is falling, check if it has dropped below the hysteresis cooldown threshold
	currIdx := h.indexOfDuty(h.currentDuty)
	if currIdx == -1 {
		h.currentDuty = standardDuty
		return h.currentDuty
	}

	stayThreshold := h.thresholds[currIdx] - h.hysteresis
	if temp >= stayThreshold {
		// Temperature is still above the cooldown threshold; keep the current higher speed
		return h.currentDuty
	}

	// Temp has dropped below the cooldown threshold; safe to drop to the lower speed
	h.currentDuty = standardDuty
	return h.currentDuty
}

// getStandardDuty returns the standard target duty cycle for a given temperature without hysteresis.
func (h *HysteresisController) getStandardDuty(temp float64) int {
	targetDuty := 0
	// Find the highest threshold that is less than or equal to the current temperature
	for i := len(h.thresholds) - 1; i >= 0; i-- {
		if temp >= h.thresholds[i] {
			targetDuty = h.dutyCycles[i]
			break
		}
	}
	return targetDuty
}

// indexOfDuty returns the index of a given duty cycle in the config array.
func (h *HysteresisController) indexOfDuty(duty int) int {
	for i, v := range h.dutyCycles {
		if v == duty {
			return i
		}
	}
	return -1
}

// GetCurrentDuty returns the currently set duty cycle.
func (h *HysteresisController) GetCurrentDuty() int {
	return h.currentDuty
}
