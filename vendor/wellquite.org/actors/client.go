package actors

import (
	"sync/atomic"

	"wellquite.org/actors/mailbox"
)

type TerminationSubscription struct {
	client  *ClientBase
	fun     func(subscription *TerminationSubscription, err error, caughtPanic interface{})
	expired uint32
}

// Cancels the subscription. If this returns true then it is
// guaranteed the callback function has not been invoked and will not
// be invoked. If this returns false then it could mean the callback
// function has already been invoked (or at least started), or it
// could mean the subscription has already been cancelled. As the
// callback function is invoked in a fresh go-routine, it is entirely
// possible for the callback function to be running concurrently with
// a call to Cancel, in which case, Cancel() can return false before
// the callback function has run to completion.
func (self *TerminationSubscription) Cancel() bool {
	if atomic.CompareAndSwapUint32(&self.expired, 0, 1) {
		// best effort - ignore the result
		self.client.Send((*terminatedSubscriptionUnregister)(self))
		return true
	}
	return false
}

func (self *TerminationSubscription) run(err error, caughtPanic interface{}) {
	if atomic.CompareAndSwapUint32(&self.expired, 0, 1) {
		self.fun(self, err, caughtPanic)
	}
}

// Low-level client-side interface to every actor.
type Client interface {
	// Post a message into the actor's mailbox. Returns true iff it was
	// possible to add the message to the actor's mailbox. There is no
	// guarantee that the actor will retrieve and process the message
	// before it terminates.
	Send(msg mailbox.Msg) (success bool)

	// Post a synchronous message into the actor's mailbox. A
	// synchronous message can be waited upon for the message to be
	// processed by the server-side of the actor.
	//
	// If waitForReply is true, then this method will block until:
	//
	//   a) the mailbox is closed before we can enqueue the message
	//      (the actor has terminated), in which case we return false.
	//
	//   b) the message is enqueued, but the actor terminates before it
	//      can process the message, in which case we return false.
	//
	//   c) the message is enqueued, retrieved, processed by the actor,
	//      and the actor marks the processing of the message as being
	//      complete, in which case we return true.
	//
	// If waitForReply is false then SendSync will return true iff it
	// is able to enqueue the message into the mailbox (case (a)
	// above). It is then up to the caller to invoke msg.WaitForReply()
	// before it accesses any reply fields in the message.
	SendSync(msg MsgSync, waitForReply bool) (success bool)

	// Request the actor terminates. Does not return until the actor
	// has terminated: the server-side Terminated method must have
	// finished before this returns. Idempotent.
	TerminateSync()

	// Creates a subscription to observe the termination of the actor.
	//
	// If the subscription cannot be created (the actor terminated
	// before the subscription could be registered) then the returned
	// value will be nil. In this case, it is guaranteed the callback
	// function will never be invoked.
	//
	// If the returned value is non-nil then it is guaranteed the
	// callback function will be invoked exactly once when the actor
	// terminates (unless the subscription is cancelled before the
	// actor terminates).
	//
	// If the callback function is invoked, it is invoked in a fresh
	// go-routine, and does not block the termination of the actor. It
	// is invoked with the exact same subscription object as the method
	// returns (which is useful for identification purposes), along
	// with the error, if any, which caused the actor to terminate.
	OnTermination(func(subscription *TerminationSubscription, err error, caughtPanic interface{})) *TerminationSubscription
}

var _ Client = (*ClientBase)(nil)

// ClientBase implements the Client interface and provides the basic
// low-level client-side functionality to send messages to your actor.
type ClientBase struct {
	mailboxWriter *mailbox.MailboxWriter
	postSendHook  func()
}

// For the Client interface.
func (self *ClientBase) Send(msg mailbox.Msg) (success bool) {
	success = self.mailboxWriter.Send(msg)
	if success {
		self.postSendHook()
	}
	return success
}

// For the Client interface.
func (self *ClientBase) SendSync(msg MsgSync, waitForReply bool) (success bool) {
	msg.initMsg(self.mailboxWriter)
	if !self.Send(msg) {
		return false
	}
	if waitForReply {
		return msg.WaitForReply()
	} else {
		return true
	}
}

type msgTerminate struct{}

// For the Client interface.
func (self *ClientBase) TerminateSync() {
	if self.Send(msgTerminate{}) {
		self.mailboxWriter.WaitForTermination()
	}
}

type terminatedSubscriptionRegister struct {
	MsgSyncBase
	subscription *TerminationSubscription
}

type terminatedSubscriptionUnregister TerminationSubscription

// For the Client interface.
func (self *ClientBase) OnTermination(fun func(subscription *TerminationSubscription, err error, caughtPanic interface{})) *TerminationSubscription {
	subscription := &TerminationSubscription{
		client: self,
		fun:    fun,
	}
	msg := &terminatedSubscriptionRegister{
		subscription: subscription,
	}
	if !self.SendSync(msg, true) {
		return nil
	}
	return subscription
}
