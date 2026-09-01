package pool

import "time"

type containerState uint8

const (
	// stateIdle = 0 is the zero value of containerState, so a Container is idle by default.
	stateIdle containerState = iota
	stateBusy
	stateDraining // being removed; Claim must skip it
)

// ID is the only exported method: the executor calls docker exec using it
// without the pool exposing any other mutable field.
type Container struct {
	id       string
	language string
	// memoryBytes is what the pool charges against its budget: the language's
	// ceiling. limitBytes is what the kernel enforces right now, never above it.
	memoryBytes int64
	limitBytes  int64
	state       containerState
	lastUsedAt  time.Time
}

func (c *Container) ID() string { return c.id }
