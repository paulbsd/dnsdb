package actors

import "wellquite.org/actors/mailbox"

// Messages that require a response can implement this interface (by
// embedding MsgSyncBase). That allows the client-side to wait to
// receive a response, and the server-side to signal when the response
// has been provided.
//
// The expectation is that message structs that embed MsgSyncBase also
// include both query and reply fields.
type MsgSync interface {
	// Used by the actor server-side; returns false iff it was already
	// marked as processed.
	//
	// Once reply fields have been set in the message, call this to
	// signal to any waiting client that the message has been processed
	// and that reply values can now be safely read.
	MarkProcessed() bool

	// Used by the actor client-side; returns true iff MarkProcessed() has
	// been called on this msg. Blocks until either MarkProcessed() is
	// called, or it is known that MarkProcessed() can never be called
	// (i.e. the server has died before processing the msg).
	WaitForReply() bool

	// Called as part of enqueuing a synchronous message.
	initMsg(mailboxWriter *mailbox.MailboxWriter)
}

var _ MsgSync = (*MsgSyncBase)(nil)

// Embed MsgSyncBase anonymously within each message which requires a
// response from an actor server.
type MsgSyncBase struct {
	waitChan       chan struct{}
	terminatedChan <-chan struct{}
}

// For the MsgSync interface.
func (self *MsgSyncBase) initMsg(mailboxWriter *mailbox.MailboxWriter) {
	self.waitChan = make(chan struct{})
	self.terminatedChan = mailboxWriter.TerminatedChan
}

// For the MsgSync interface.
func (self *MsgSyncBase) MarkProcessed() bool {
	select {
	case <-self.waitChan:
		return false
	default:
		close(self.waitChan)
		return true
	}
}

// For the MsgSync interface.
func (self *MsgSyncBase) WaitForReply() bool {
	select {
	case <-self.waitChan:
		return true
	case <-self.terminatedChan:
		// if they were both ready to recieve on, we could end up here
		// (it's random), so we need to test again to see whether
		// WaitChan can be received on:
		select {
		case <-self.waitChan:
			return true
		default:
			return false
		}
	}
}
