// Every Mailbox is safe for multiple concurrent writers, and one reader. It is unbounded in length.
package mailbox

import (
	"sync"
)

const InitialMsgChanLen = 16

type Msg interface{}

// We form a linked-list of MsgChanCells.
type MsgChanCell struct {
	// the lock protects next only
	lock    sync.Mutex
	next    *MsgChanCell
	channel chan Msg
}

// do not attempt to access the same reader from >1 go-routine
type MailboxReader struct {
	cell           *MsgChanCell
	terminatedChan chan struct{}
}

// For use by the reader go-routine only.
func (self *MailboxReader) Receive() Msg {
	for {
		currentCell := self.cell
		if msg, ok := <-currentCell.channel; ok {
			return msg
		}
		currentCell.lock.Lock()
		self.cell = currentCell.next
		currentCell.lock.Unlock()
	}
}

// For use by the reader go-routine only
func (self *MailboxReader) Length() (length int) {
	currentCell := self.cell
	for currentCell != nil {
		length += len(currentCell.channel)
		currentCell.lock.Lock()
		nextCell := currentCell.next
		currentCell.lock.Unlock()
		currentCell = nextCell
	}
	return length
}

// For use by the reader go-routine only. Returns true iff the mailbox
// is currently empty.
func (self *MailboxReader) IsEmpty() (empty bool) {
	if len(self.cell.channel) == 0 {
		self.cell.lock.Lock()
		empty = self.cell.next == nil
		self.cell.lock.Unlock()
		return empty
	} else {
		return false
	}
}

// Used by the reader to indicate the reader has exited. It is
// idempotent.
func (self *MailboxReader) Terminate() {
	select {
	case <-self.terminatedChan:
	default:
		close(self.terminatedChan)
	}
}

// a writer can be safely used across multiple go routines
type MailboxWriter struct {
	// the lock protects cell only
	lock           sync.RWMutex
	cell           *MsgChanCell
	chanLen        int
	TerminatedChan <-chan struct{}
}

// returns false if the channel has been terminated. true indicates
// the msg has been sent, not that it's going to be received or
// processed.
func (self *MailboxWriter) Send(msg Msg) (success bool) {
	for {
		select {
		case <-self.TerminatedChan:
			return false

		default:
			self.lock.RLock()
			currentCell := self.cell

			select {
			case currentCell.channel <- msg:
				self.lock.RUnlock()
				return true

			default:
				self.lock.RUnlock()
				self.lock.Lock()
				if self.cell == currentCell {
					nextCell := &MsgChanCell{
						channel: make(chan Msg, self.chanLen),
					}
					currentCell.lock.Lock()
					currentCell.next = nextCell
					currentCell.lock.Unlock()

					close(currentCell.channel)

					self.cell = nextCell
					self.chanLen *= 2
				}
				self.lock.Unlock()
			}
		}
	}
}

func (self *MailboxWriter) WaitForTermination() {
	<-self.TerminatedChan
}

func NewMailbox() (writer *MailboxWriter, reader *MailboxReader) {
	channel := make(chan Msg, InitialMsgChanLen)
	terminatedChan := make(chan struct{})
	currentCell := &MsgChanCell{
		channel: channel,
	}

	writer = &MailboxWriter{
		cell:           currentCell,
		chanLen:        InitialMsgChanLen * 2,
		TerminatedChan: terminatedChan,
	}
	reader = &MailboxReader{
		cell:           currentCell,
		terminatedChan: terminatedChan,
	}
	return
}
