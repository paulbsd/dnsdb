package actors

import "wellquite.org/actors/mailbox"

type FiniteStateMachineState interface {
	// Enter is called immediately as soon as this state becomes the
	// current state for the FiniteStateMachine. It is only called if
	// the previous current state was different.
	Enter() (err error)

	// HandleMsg serves the same purpose as Server.HandleMsg, only it's
	// extended here to allow the nextState to be set. You are allowed
	// to return nil as the nextState, which is interpreted as
	// no-change.
	HandleMsg(msg mailbox.Msg) (nextState FiniteStateMachineState, err error)
}

// A FiniteStateMachine is an actor server which forwards messages to
// its current state (a FiniteStateMachineState). The current state
// can return a nextState. Or, the current state can also explicitly
// call Become in order to set the next state.
//
// Whenever the state changes, for the new state, Enter is called
// immediately.
type FiniteStateMachine struct {
	Server
	state FiniteStateMachineState
}

var _ Server = (*FiniteStateMachine)(nil)

// Sets the FSM's state to be nextState. If nextState is non nil and
// different to the current state, returns the result of
// nextState.Enter().
func (self *FiniteStateMachine) Become(nextState FiniteStateMachineState) (err error) {
	if self.state == nextState || nextState == nil {
		return nil
	}
	self.state = nextState
	return nextState.Enter()
}

// For the Server interface.
//
// Calls HandleMsg on the FSM's state. If that handler returns a nil
// error, returns the result of Become(nextState).
func (self *FiniteStateMachine) HandleMsg(msg mailbox.Msg) (err error) {
	nextState, err := self.state.HandleMsg(msg)
	if err != nil {
		return err
	}
	return self.Become(nextState)
}
