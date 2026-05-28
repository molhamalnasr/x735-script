package gpio

import (
	"fmt"
	"sync"

	"github.com/warthog618/go-gpiocdev"
)

// GPIOManager handles operations on Linux GPIO character devices.
type GPIOManager struct {
	chipName       string
	requestedLines map[int]*gpiocdev.Line
	mu             sync.Mutex
}

// NewGPIOManager creates a new GPIOManager instance for a given chip index.
func NewGPIOManager(chipIndex int) *GPIOManager {
	return &GPIOManager{
		chipName:       fmt.Sprintf("gpiochip%d", chipIndex),
		requestedLines: make(map[int]*gpiocdev.Line),
	}
}

// RequestInput configures a pin as input.
func (m *GPIOManager) RequestInput(pin int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.requestedLines[pin]; exists {
		return nil
	}

	line, err := gpiocdev.RequestLine(m.chipName, pin, gpiocdev.AsInput)
	if err != nil {
		return fmt.Errorf("failed to request input line %d on %s: %w", pin, m.chipName, err)
	}

	m.requestedLines[pin] = line
	return nil
}

// RequestOutput configures a pin as output with an initial value (0 or 1).
func (m *GPIOManager) RequestOutput(pin int, initialValue int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if line, exists := m.requestedLines[pin]; exists {
		line.Close()
		delete(m.requestedLines, pin)
	}

	line, err := gpiocdev.RequestLine(m.chipName, pin, gpiocdev.AsOutput(initialValue))
	if err != nil {
		return fmt.Errorf("failed to request output line %d on %s: %w", pin, m.chipName, err)
	}

	m.requestedLines[pin] = line
	return nil
}

// SetValue sets the state of a requested output pin (0 or 1).
func (m *GPIOManager) SetValue(pin int, val int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	line, exists := m.requestedLines[pin]
	if !exists {
		return fmt.Errorf("pin %d has not been requested as output", pin)
	}

	err := line.SetValue(val)
	if err != nil {
		return fmt.Errorf("failed to set value for pin %d: %w", pin, err)
	}

	return nil
}

// GetValue reads the state of an input pin.
func (m *GPIOManager) GetValue(pin int) (int, error) {
	m.mu.Lock()
	line, exists := m.requestedLines[pin]
	m.mu.Unlock()

	if !exists {
		err := m.RequestInput(pin)
		if err != nil {
			return 0, err
		}
		m.mu.Lock()
		line = m.requestedLines[pin]
		m.mu.Unlock()
	}

	val, err := line.Value()
	if err != nil {
		return 0, fmt.Errorf("failed to read value for pin %d: %w", pin, err)
	}

	return val, nil
}

// WatchEdge registers a handler for edge transitions on an input pin.
func (m *GPIOManager) WatchEdge(pin int, handler func(gpiocdev.LineEvent)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if line, exists := m.requestedLines[pin]; exists {
		line.Close()
		delete(m.requestedLines, pin)
	}

	line, err := gpiocdev.RequestLine(m.chipName, pin,
		gpiocdev.AsInput,
		gpiocdev.WithEventHandler(handler),
		gpiocdev.WithBothEdges,
	)
	if err != nil {
		return fmt.Errorf("failed to request watch line %d on %s: %w", pin, m.chipName, err)
	}

	m.requestedLines[pin] = line
	return nil
}

// Close releases all requested lines.
func (m *GPIOManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for pin, line := range m.requestedLines {
		line.Close()
		delete(m.requestedLines, pin)
	}
}
