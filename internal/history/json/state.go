package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// TransactionState represents the state of a transaction
type TransactionState string

const (
	// StateInitial is the initial state of a transaction
	StateInitial TransactionState = "initial"

	// StatePrepared indicates the transaction is prepared and ready to commit
	StatePrepared TransactionState = "prepared"

	// StateCommitted indicates the transaction has been successfully committed
	StateCommitted TransactionState = "committed"

	// StateRolledBack indicates the transaction has been rolled back
	StateRolledBack TransactionState = "rolled_back"

	// StateFailed indicates the transaction has failed
	StateFailed TransactionState = "failed"
)

var (
	// ErrInvalidStateTransition is returned when an invalid state transition is attempted
	ErrInvalidStateTransition = errors.New("invalid state transition")

	// ErrTransactionCompleted is returned when attempting to modify a completed transaction
	ErrTransactionCompleted = errors.New("transaction already completed")
)

// TransactionMetadata contains metadata about the transaction
type TransactionMetadata struct {
	State       TransactionState `json:"state"`
	StartTime   time.Time        `json:"start_time"`
	UpdateTime  time.Time        `json:"update_time"`
	Description string           `json:"description"`
	Error       string           `json:"error,omitempty"`
}

// canTransitionTo checks if the current state can transition to the target state
func (s TransactionState) canTransitionTo(target TransactionState) bool {
	allowedTransitions := map[TransactionState][]TransactionState{
		StateInitial: {
			StatePrepared,
			StateFailed,
		},
		StatePrepared: {
			StateCommitted,
			StateRolledBack,
			StateFailed,
		},
		StateCommitted:  {}, // Final state
		StateRolledBack: {}, // Final state
		StateFailed:     {}, // Final state
	}

	allowed, exists := allowedTransitions[s]
	if !exists {
		return false
	}

	for _, allowedState := range allowed {
		if allowedState == target {
			return true
		}
	}
	return false
}

// IsTerminal returns true if the state is a terminal state
func (s TransactionState) IsTerminal() bool {
	switch s {
	case StateCommitted, StateRolledBack, StateFailed:
		return true
	default:
		return false
	}
}

// String returns the string representation of the state
func (s TransactionState) String() string {
	return string(s)
}

// MarshalJSON implements json.Marshaler
func (s TransactionState) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

// UnmarshalJSON implements json.Unmarshaler
func (s *TransactionState) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	switch TransactionState(str) {
	case StateInitial, StatePrepared, StateCommitted, StateRolledBack, StateFailed:
		*s = TransactionState(str)
		return nil
	default:
		return fmt.Errorf("invalid transaction state: %s", str)
	}
}

// NewTransactionMetadata creates a new transaction metadata
func NewTransactionMetadata(description string) TransactionMetadata {
	now := time.Now()
	return TransactionMetadata{
		State:       StateInitial,
		StartTime:   now,
		UpdateTime:  now,
		Description: description,
	}
}

// Transition attempts to transition the transaction to a new state
func (m *TransactionMetadata) Transition(target TransactionState) error {
	if m.State.IsTerminal() {
		return ErrTransactionCompleted
	}

	if !m.State.canTransitionTo(target) {
		return fmt.Errorf("%w: cannot transition from %s to %s",
			ErrInvalidStateTransition, m.State, target)
	}

	m.State = target
	m.UpdateTime = time.Now()
	return nil
}

// SetError sets the error message and transitions to failed state
func (m *TransactionMetadata) SetError(err error) error {
	if err := m.Transition(StateFailed); err != nil {
		return err
	}
	m.Error = err.Error()
	return nil
}

// Duration returns the duration since the transaction started
func (m *TransactionMetadata) Duration() time.Duration {
	return m.UpdateTime.Sub(m.StartTime)
}
