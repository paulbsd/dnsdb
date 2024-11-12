# Actors

The purpose of this package is to allow you to create
[actors](https://en.wikipedia.org/wiki/Actor_model) in Go.

An [introduction and
tutorial](https://wellquite.org/posts/lets_build/edist_actors/) to
using this library is available.

Think of an actor like a little server: it is created, it receives
messages and processes them, and it terminates. As part of processing
a message, it can reply to the message, it can send messages to other
actors that it knows of, it can create new actors, and it can
terminate.

Each actor has a mailbox. Clients of the actor call the actor's public
API, which will typically post messages into this mailbox. The actor
will see that there are messages in the mailbox, retrieve each
message, and process them. Whilst posting a message into a mailbox is
mainly an asynchronous activity, the message itself may require a
reply. The sender can choose to block and wait for the reply at any
time. This functionality is provided by the
[MsgSync](https://pkg.go.dev/wellquite.org/actors#MsgSync) and
[MsgSyncBase](https://pkg.go.dev/wellquite.org/actors#MsgSyncBase)
types.

Each actor-server is a Go-routine. The mailbox is implemented by
chans. It is safe for multiple concurrent Go-routines to post messages
to the same actor, and it is always the case that an actor-server
itself is single-threaded - a single Go-routine. If you wish for more
elaborate setups then that is for you: some sort of sharding or
session management across a set of actors of the same type is
perfectly possible, but is not currently provided directly by this
package.

Use different struct types to implement the client-side and the
server-side of each actor. The client-side should embed (probably
anonymously) an implementation of the
[Client](https://pkg.go.dev/wellquite.org/actors#Client) type. The
server-side should anonymously embed an implementation of the
[Server](https://pkg.go.dev/wellquite.org/actors#Server) type (either
[BackPressureServerBase](https://pkg.go.dev/wellquite.org/actors#BackPressureServerBase)
or [ServerBase](https://pkg.go.dev/wellquite.org/actors#ServerBase)).

The server-side of each actor must satisfy the
[Server](https://pkg.go.dev/wellquite.org/actors#Server) interface,
and so has to provide three methods: `Init`, `HandleMsg`, and
`Terminated`. [BackPressureServerBase](https://pkg.go.dev/wellquite.org/actors#BackPressureServerBase)
and [ServerBase](https://pkg.go.dev/wellquite.org/actors#ServerBase)
have implementations of all of these, so you need only reimplement the
ones you need. For all methods that you implement yourself on the
server-side, make certain that you also call the embedded version. If
you fail to do this for `Init`, then various parts of the actor will
not be set up properly. If you fail to do this for `HandleMsg` then
the actor will not be able to be terminate, and if you fail to do this
for `Terminated`, then termination subscriptions will not function
correctly.

`Init` is the first method invoked by the new actor, and is called
synchronously as part of spawning the new actor. I.e. at the point
that `Init` is invoked (in the new actor's Go-routine), `Spawn` will
not have returned, and no one will yet know of the existence of this
new actor. So, if the new actor decides to send messages to itself, it
can guarantee that those messages will be the first items in its own
mailbox. This is useful for being able to do deferred asynchronous
initialisation, for example. If `Init` fails (returns a non-nil
error), then `Spawn` fails too, and returns the same error.

`HandleMsg` is invoked for each message that is received from the
actor's mailbox. `HandleMsg` returns an error; if the error is non-nil
then the actor terminates: the actor's mailbox is closed, `Terminated`
is invoked, and finally the actor's Go-routine exits. All remaining
messages in the actor's mailbox are discarded, and anyone else waiting
on responses from the actor will be informed that the actor has
terminated and no further responses will be forthcoming.

An [introduction and
tutorial](https://wellquite.org/posts/lets_build/edist_actors/) to
using this library is available.
