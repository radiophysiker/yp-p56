package order

import (
	"fmt"
)

// StateTransition defines a possible transition between states
type StateTransition struct {
	From Status
	To   Status
}

// StateMachine manages transitions between order states
type StateMachine struct {
	validTransitions map[StateTransition]bool
}

// NewStateMachine creates a new state machine with predefined transition rules
func NewStateMachine() *StateMachine {
	sm := &StateMachine{
		validTransitions: make(map[StateTransition]bool),
	}

	validTransitions := []StateTransition{
		{From: StatusNew, To: StatusProcessing},
		{From: StatusNew, To: StatusInvalid},

		{From: StatusProcessing, To: StatusProcessed},
		{From: StatusProcessing, To: StatusInvalid},
		{From: StatusProcessing, To: StatusNew},
	}

	for _, transition := range validTransitions {
		sm.validTransitions[transition] = true
	}

	return sm
}

// CanTransition checks whether a transition from one state to another is possible
func (sm *StateMachine) CanTransition(from, to Status) bool {
	if from == to {
		return false
	}

	transition := StateTransition{From: from, To: to}
	return sm.validTransitions[transition]
}

// ValidateTransition checks the possibility of a transition and returns an error if the transition is invalid
func (sm *StateMachine) ValidateTransition(from, to Status) error {
	if !sm.CanTransition(from, to) {
		return &InvalidTransitionError{
			From: from,
			To:   to,
		}
	}
	return nil
}

// GetValidTransitions returns a list of possible states to transition to from the current state
func (sm *StateMachine) GetValidTransitions(from Status) []Status {
	var validStates []Status

	for transition := range sm.validTransitions {
		if transition.From == from {
			validStates = append(validStates, transition.To)
		}
	}

	return validStates
}

// IsFinalState checks whether a state is final (i.e., no transitions are possible from it)
func (sm *StateMachine) IsFinalState(status Status) bool {
	for transition := range sm.validTransitions {
		if transition.From == status {
			return false
		}
	}
	return true
}

// InvalidTransitionError represents an error for an invalid state transition
type InvalidTransitionError struct {
	From Status
	To   Status
}

// Error implements the error interface for InvalidTransitionError
func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("invalid state transition from %s to %s", e.From, e.To)
}

// IsInvalidTransitionError checks whether an error is an InvalidTransitionError
func IsInvalidTransitionError(err error) bool {
	_, ok := err.(*InvalidTransitionError)
	return ok
}
