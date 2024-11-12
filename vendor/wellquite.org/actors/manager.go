package actors

import (
	"errors"
	"sync"

	"github.com/rs/zerolog"
	"wellquite.org/actors/mailbox"
)

// ManagerServerBase is the server-side for a manager actor. ManagerClientBase is the client-side.
type ManagerServerBase struct {
	BackPressureServerBase
	// The child actors of this manager.
	Children map[*TerminationSubscription]Client
}

var _ Server = (*ManagerServerBase)(nil)

// For the Server interface.
func (self *ManagerServerBase) Init(log zerolog.Logger, mailboxReader *mailbox.MailboxReader, selfClient *ClientBase) (err error) {
	self.Children = make(map[*TerminationSubscription]Client)
	return self.BackPressureServerBase.Init(log, mailboxReader, selfClient)
}

// For the Server interface.
//
// Understands spawn messages, and childTerminated messages.
func (self *ManagerServerBase) HandleMsg(msg mailbox.Msg) (err error) {
	switch msgT := msg.(type) {
	case *managerSpawnMsg:
		clientBase, err := Spawn(self.Log, msgT.server, msgT.name)
		if err == nil {
			msgT.clientBase = clientBase
			if subscription := clientBase.OnTermination(self.childTerminated); subscription != nil {
				self.Children[subscription] = clientBase
			}
			// consider: the new actor could, in its init, have sent
			// itself a message which it has now received and processed,
			// that causes it to terminate. So it is not guaranteed that
			// this subscription will be created.

		} else {
			msgT.initErr = err
		}
		// Make sure we set up the subscription *before* we inform our
		// client that the actor's been spawned.
		msgT.MarkProcessed()
		return nil

	case *childTerminated:
		if _, found := self.Children[msgT.subscription]; found {
			delete(self.Children, msgT.subscription)
			// default policy is that if children terminate "normally"
			// then that's ok. But if any of them exit abnormally then we
			// tear ourself and all our children down.
			if msgT.caughtPanic != nil {
				panic(msgT.caughtPanic)
			} else if msgT.err != ErrNormalActorTermination {
				return msgT.err
			}
		}
		return nil

	default:
		return self.BackPressureServerBase.HandleMsg(msg)
	}
}

// For the Server interface.
//
// Ensures all child actors of the manager are terminated. Termination
// of all child actors happens concurrently, but this method blocks
// until all child actors have terminated.
func (self *ManagerServerBase) Terminated(err error, caughtPanic interface{}) {
	if len(self.Children) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(self.Children))
		for subscription, child := range self.Children {
			subscription.Cancel()
			childCopy := child
			go func() {
				defer wg.Done()
				childCopy.TerminateSync()
			}()
		}
		wg.Wait()
	}

	self.BackPressureServerBase.Terminated(err, caughtPanic)
}

type childTerminated struct {
	subscription *TerminationSubscription
	err          error
	caughtPanic  interface{}
}

func (self *ManagerServerBase) childTerminated(subscription *TerminationSubscription, err error, caughtPanic interface{}) {
	self.SelfClient.Send(&childTerminated{
		subscription: subscription,
		err:          err,
		caughtPanic:  caughtPanic,
	})
}

// Managers exist to support the management of child actors.
//
// • If a child actor terminates with ErrNormalActorTermination
// (normal termination) then the manager and all its other children
// continue to work.
//
// • If a child actor terminates for any other reason (abnormal
// termination) then the manager actor itself terminates.
//
// • Whenever the manager terminates, it makes sure that all its child
// actors have terminated.
//
// • Because TerminateSync is synchronous, calling TerminateSync on a
// manager will not return until all its children have also fully
// terminated too.
type ManagerClient interface {
	Client
	// Spawn a new actor based on the provided Server and name. The new
	// actor is a child of the manager.
	Spawn(server Server, name string) (*ClientBase, error)
}

// ManagerClientBase implements the ManagerClient interface.
type ManagerClientBase struct {
	*ClientBase
}

var _ ManagerClient = (*ManagerClientBase)(nil)

func SpawnManager(log zerolog.Logger, name string) (*ManagerClientBase, error) {
	clientBase, err := Spawn(log, &ManagerServerBase{}, name)
	if err != nil {
		return nil, err
	}
	return &ManagerClientBase{ClientBase: clientBase}, nil
}

type managerSpawnMsg struct {
	MsgSyncBase
	server     Server
	name       string
	clientBase *ClientBase
	initErr    error
}

// For the ManagerClient interface.
//
// Synchronously spawns a new actor as a child of the manager. If the
// manager is not terminated then the error returned is the result of
// the new actor's Init method.
func (self *ManagerClientBase) Spawn(server Server, name string) (*ClientBase, error) {
	msg := &managerSpawnMsg{
		server: server,
		name:   name,
	}
	if self.SendSync(msg, true) {
		return msg.clientBase, msg.initErr
	}
	return nil, errors.New("manager is terminated")
}
