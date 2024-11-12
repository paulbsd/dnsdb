package actors

import (
	"wellquite.org/actors/mailbox"
)

// This is deliberately a very simple implementation of back pressure:
// every 1000 sends from each client, we force the client to do a
// round-trip to the server. If the server is overloaded, this will
// force clients to block and wait.
//
// It is almost certainly possible to do something more sophisticated,
// probably with ingress and egress rates, and trying to minimise
// queue lengths to optimise for latency. But I couldn't make it work,
// so, KISS.
//
// You can use BackPressureServerBase in place of ServerBase. If you
// do so, you must also use BackPressureClientBaseFactory in place of
// ClientBase.
type BackPressureServerBase struct {
	ServerBase
}

const QuotaSize = 1000

var _ Server = (*BackPressureServerBase)(nil)

// For the Server interface.
//
// Understands quota messages.
func (self *BackPressureServerBase) HandleMsg(msg mailbox.Msg) (err error) {
	if quota, isQuota := msg.(*quota); isQuota {
		quota.size = QuotaSize
		quota.MarkProcessed()
		return nil
	}
	return self.ServerBase.HandleMsg(msg)
}

type quota struct {
	MsgSyncBase
	size int
}

// BackPressureClientBaseFactory must be used together with
// BackPressureServerBase. It allocates a quota to each client, which
// means each client now carries some mutable state. For this reason,
// the client-side should follow a factory pattern: i.e. the
// ClientBase that is returned from Spawn() should not be used
// directly; instead pass it to NewBackPressureClientBaseFactory and
// call NewClient() on that.
type BackPressureClientBaseFactory struct {
	client *ClientBase
}

func NewBackPressureClientBaseFactory(client *ClientBase) *BackPressureClientBaseFactory {
	return &BackPressureClientBaseFactory{
		client: client,
	}
}

func (self *BackPressureClientBaseFactory) NewClient() *BackPressureClientBase {
	return &BackPressureClientBase{
		ClientBase: self.client,
		quota:      quota{size: 1},
	}
}

type BackPressureClientBase struct {
	*ClientBase
	quota     quota
	sendCount int
}

var _ Client = (*BackPressureClientBase)(nil)

func (self *BackPressureClientBase) ensureQuota() (success bool) {
	self.sendCount += 1
	if self.sendCount <= self.quota.size {
		return true
	}
	success = self.ClientBase.SendSync(&self.quota, true)
	if success {
		self.sendCount = 0
	}
	return success
}

// Same as ClientBase.Send() - i.e. posts the message to the actor's
// mailbox. But does additional work to obey the current back-pressure
// mechanism.
func (self *BackPressureClientBase) Send(msg mailbox.Msg) (success bool) {
	if !self.ensureQuota() {
		return false
	}
	return self.ClientBase.Send(msg)
}

// Same as ClientBase.SendSync() - i.e. posts a MsgSync message to the
// actor's mailbox and optionally waits for the server to reply or
// terminate. But does additional work to obey the current
// back-pressure mechanism.
func (self *BackPressureClientBase) SendSync(msg MsgSync, waitForReply bool) (success bool) {
	if !self.ensureQuota() {
		return false
	}
	return self.ClientBase.SendSync(msg, waitForReply)
}
