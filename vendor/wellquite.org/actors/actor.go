package actors

import (
	"errors"
	"runtime/debug"

	"github.com/rs/zerolog"
	"wellquite.org/actors/mailbox"
)

// Synchronously creates a new actor. The error returned is the error
// from the new actor's server-side Init method.
func Spawn(log zerolog.Logger, server Server, name string) (*ClientBase, error) {
	return SpawnWithExtraHooks(log, nil, server, name)
}

func SpawnWithExtraHooks(log zerolog.Logger, hooks ActorHooks, server Server, name string) (*ClientBase, error) {
	if name != "" {
		log = log.With().Str("actor", name).Logger()
	}
	writer, reader := mailbox.NewMailbox()
	initResultChan := make(chan error)

	preReceiveHook, postSendHook := hooks.unzip()
	clientBase := &ClientBase{
		mailboxWriter: writer,
		postSendHook:  postSendHook,
	}

	go loop(log, preReceiveHook, server, reader, clientBase, initResultChan)

	err := <-initResultChan
	if err != nil {
		return nil, err
	}

	return clientBase, nil
}

func loop(log zerolog.Logger, preReceiveHook func(), server Server, mailboxReader *mailbox.MailboxReader, client *ClientBase, initResultChan chan error) {
	var err error
	defer func() {
		caughtPanic := recover()
		// Use a defer here too, so that even if Terminated panics we'll
		// still log, and Terminate the mailbox (which is critical for
		// releasing waiting messages).
		defer func() {
			if caughtPanic != nil {
				// Caller(0) is this defer
				// Caller(1) is the parent defer
				// Caller(2) is the builtin panic function
				// so we want Caller(3)
				// We also use .WithLevel(zerolog.PanicLevel) rather than .Panic() because the latter actually panics after logging which we don't want.
				log.WithLevel(zerolog.PanicLevel).Str("status", "terminated").Caller(3).Interface("panic", caughtPanic).Msg(string(debug.Stack()))
			} else if err != nil && err != ErrNormalActorTermination {
				log.Error().Str("status", "terminated").Err(err).Send()
			} else if log.Debug().Enabled() {
				log.Debug().Str("status", "terminated").Send()
			}
			mailboxReader.Terminate()
		}()
		server.Terminated(err, caughtPanic)
	}()

	if log.Debug().Enabled() {
		log.Debug().Str("status", "init").Send()
	}
	err = server.Init(log, mailboxReader, client)
	initResultChan <- err

	if err != nil {
		if log.Debug().Enabled() {
			log.Debug().Str("status", "init").Err(err).Send()
		}

	} else {
		if log.Debug().Enabled() {
			log.Debug().Str("status", "ready").Send()
		}

		for err == nil {
			preReceiveHook()
			msg := mailboxReader.Receive()
			err = server.HandleMsg(msg)
		}
	}
}

type ActorHook struct {
	PreReceiveHook func()
	PostSendHook   func()
}

type ActorHooks []ActorHook

func (self ActorHooks) unzip() (preReceiveHook, postSendHook func()) {
	preReceiveHooks := make([]func(), 0, len(self))
	postSendHooks := make([]func(), 0, len(self))
	for _, hook := range self {
		if hook.PreReceiveHook != nil {
			preReceiveHooks = append(preReceiveHooks, hook.PreReceiveHook)
		}
		if hook.PostSendHook != nil {
			postSendHooks = append(postSendHooks, hook.PostSendHook)
		}
	}

	switch len(preReceiveHooks) {
	case 0:
		preReceiveHook = func() {}
	case 1:
		preReceiveHook = preReceiveHooks[0]
	default:
		preReceiveHook = func() {
			for _, hook := range preReceiveHooks {
				hook()
			}
		}
	}

	switch len(postSendHooks) {
	case 0:
		postSendHook = func() {}
	case 1:
		postSendHook = postSendHooks[0]
	default:
		postSendHook = func() {
			for _, hook := range postSendHooks {
				hook()
			}
		}
	}

	return preReceiveHook, postSendHook
}

// Use this error as the return value from any actor server-side
// call-back in order to indicate the actor should terminate in a
// normal way. I.e. no real error has occurred, the actor just wishes
// to terminate.
var ErrNormalActorTermination = errors.New("Normal Actor Termination")
