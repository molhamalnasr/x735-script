package power

import "time"

// Action represents the power management action to execute.
type Action string

const (
	ActionNone     Action = "none"
	ActionReboot   Action = "reboot"
	ActionPowerOff Action = "poweroff"
)

// PulseClassifier classifies the duration of a button press (pulse)
// into a specific power action (Reboot, PowerOff, or None).
type PulseClassifier struct {
	rebootMinMs int
	rebootMaxMs int
}

// NewPulseClassifier creates a new PulseClassifier instance.
func NewPulseClassifier(rebootMinMs, rebootMaxMs int) *PulseClassifier {
	return &PulseClassifier{
		rebootMinMs: rebootMinMs,
		rebootMaxMs: rebootMaxMs,
	}
}

// Classify takes the elapsed button hold duration and determines the appropriate action.
func (p *PulseClassifier) Classify(elapsed time.Duration) Action {
	elapsedMs := elapsed.Milliseconds()

	if elapsedMs > int64(p.rebootMaxMs) {
		return ActionPowerOff
	}
	if elapsedMs > int64(p.rebootMinMs) {
		return ActionReboot
	}
	return ActionNone
}
