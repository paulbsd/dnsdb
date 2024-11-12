package actors

import (
	"fmt"

	"github.com/rs/zerolog"
	"wellquite.org/actors/mailbox"
)

// Low-level server-side interface to every actor.
type Server interface {
	// This is called by the actor's new Go-routine once it's up and
	// running. If Init() returns a non-nil err, then the actor
	// terminates, and the error will be returned as the result of
	// Spawn. Spawn will block until Init() completes.
	Init(log zerolog.Logger, mailboxReader *mailbox.MailboxReader, selfClient *ClientBase) (err error)
	// Called by the actor's Go-routine for each message received from
	// its mailbox. If HandleMsg() returns a non-nil error then the
	// actor terminates.
	HandleMsg(msg mailbox.Msg) (err error)
	// Called whenever the actor terminates.
	Terminated(err error, caughtPanic interface{})

	Base() *ServerBase
}

var _ Server = (*ServerBase)(nil)

// Embed ServerBase (or BackPressureServerBase) within the struct for
// the server-side of your actors.
type ServerBase struct {
	Log           zerolog.Logger
	MailboxReader *mailbox.MailboxReader
	SelfClient    *ClientBase
	onTerminated  map[*TerminationSubscription]struct{}
}

// For the Server interface.
//
// Sets the ChanReader and SelfClient fields.
func (self *ServerBase) Init(log zerolog.Logger, mailboxReader *mailbox.MailboxReader, selfClient *ClientBase) (err error) {
	self.Log = log
	self.MailboxReader = mailboxReader
	self.SelfClient = selfClient
	return nil
}

// For the Server interface.
//
// Understands termination subscriptions, and terminate messages.
func (self *ServerBase) HandleMsg(msg mailbox.Msg) (err error) {
	switch msgT := msg.(type) {
	case *terminatedSubscriptionRegister:
		self.registerOnTermination(msgT)
	case *terminatedSubscriptionUnregister:
		self.unregisterOnTermination(msgT)
	case msgTerminate:
		err = ErrNormalActorTermination
	default:
		panic(fmt.Sprintf("Actor received unexpected message: %#v", msg))
	}
	return
}

func (self *ServerBase) registerOnTermination(registerMsg *terminatedSubscriptionRegister) {
	registerMsg.MarkProcessed()
	if self.onTerminated == nil {
		self.onTerminated = make(map[*TerminationSubscription]struct{})
	}
	self.onTerminated[registerMsg.subscription] = struct{}{}
}

func (self *ServerBase) unregisterOnTermination(subscription *terminatedSubscriptionUnregister) {
	delete(self.onTerminated, (*TerminationSubscription)(subscription))
}

// For the Server interface.
//
// Fires all termination subscriptions in new go-routines. Does not
// wait for them to complete. Does not panic or repanic regardless of
// caughtPanic.
func (self *ServerBase) Terminated(err error, caughtPanic interface{}) {
	if len(self.onTerminated) > 0 {
		for subscription := range self.onTerminated {
			subscriptionCopy := subscription
			go func() {
				subscriptionCopy.run(err, caughtPanic)
			}()
		}
	}
}

// For the Server interface.
//
// Provides access to self.
func (self *ServerBase) Base() *ServerBase {
	return self
}
