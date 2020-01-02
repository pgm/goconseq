package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleExecution(t *testing.T) {
	e := NewExecutionPlan()
	e.AddDependency(InitialState, "a", false)
	e.AddDependency("a", "b", false)

	// simulate
	e.Completed(InitialState)
	next := e.GetPrioritizedNext()
	assert.Equal(t, []string{"a"}, next)
	e.Started("a")
	next = e.GetNext()
	assert.Equal(t, []string{}, next)

	e.Completed("a")
	next = e.GetPrioritizedNext()
	assert.Equal(t, next, []string{"b"})
	e.Started("b")
	next = e.GetNext()
	assert.Equal(t, next, []string{})

	e.Completed("b")
	next = e.GetPrioritizedNext()
	assert.Equal(t, next, []string{})
	next = e.GetNext()
	assert.Equal(t, next, []string{})
}

func TestExecutionWithMultipleChildren(t *testing.T) {
	// identical to TestSimpleExecution, but running a results in two executions of a and therefore two of b
	e := NewExecutionPlan()
	e.AddDependency(InitialState, "a", false)
	e.AddDependency("a", "b", false)

	// simulate
	e.Completed(InitialState)
	next := e.GetPrioritizedNext()
	assert.Equal(t, []string{"a"}, next)
	e.Started("a")
	e.Started("a")
	next = e.GetNext()
	assert.Equal(t, []string{}, next)

	e.Completed("a")
	next = e.GetPrioritizedNext()
	assert.Equal(t, next, []string{"b"})
	e.Started("b")
	next = e.GetNext()
	assert.Equal(t, next, []string{})

	e.Completed("a")
	next = e.GetPrioritizedNext()
	assert.Equal(t, next, []string{"b"})
	e.Started("b")
	next = e.GetNext()
	assert.Equal(t, next, []string{})

	e.Completed("b")
	next = e.GetPrioritizedNext()
	assert.Equal(t, next, []string{})
	next = e.GetNext()
	assert.Equal(t, next, []string{})

	e.Completed("b")
	next = e.GetPrioritizedNext()
	assert.Equal(t, next, []string{})
	next = e.GetNext()
	assert.Equal(t, next, []string{})
}

func TestBrokenChain(t *testing.T) {
	// identical to TestSimpleExecution, but attempt to run A fails, so no attempt to run B
	e := NewExecutionPlan()
	e.AddDependency(InitialState, "a", false)
	e.AddDependency("a", "b", false)

	// simulate
	e.Completed(InitialState)
	next := e.GetPrioritizedNext()
	assert.Equal(t, []string{"a"}, next)
	next = e.GetNext()
	assert.Equal(t, []string{}, next)

	assert.True(t, e.Done())
}

// func TestWaitForAll(t *testing.T) {
// 	// identical to TestExecutionWithMultipleChildren, b waits for all A to complete
// 	e := NewExecutionPlan()
// 	e.AddDependency(InitialState, "a", false)
// 	e.AddDependency("a", "b", true)

// 	// simulate
// 	e.Completed(InitialState)
// 	next := e.GetPrioritizedNext()
// 	assert.Equal(t, []string{"a"}, next)
// 	e.Started("a")
// 	e.Started("a")
// 	next = e.GetNext()
// 	assert.Equal(t, []string{}, next)

// 	e.Completed("a")
// 	next = e.GetPrioritizedNext()
// 	assert.Equal(t, next, []string{})
// 	next = e.GetNext()
// 	assert.Equal(t, next, []string{})

// 	e.Completed("a")
// 	next = e.GetPrioritizedNext()
// 	assert.Equal(t, next, []string{"b"})
// 	e.Started("b")
// 	next = e.GetNext()
// 	assert.Equal(t, next, []string{})

// 	e.Completed("b")
// 	next = e.GetPrioritizedNext()
// 	assert.Equal(t, []string{}, next)
// 	next = e.GetNext()
// 	assert.Equal(t, []string{}, next)

// 	assert.True(t, e.Done())
// }

// // this test fails because b would need to be added to the pending list after InitialState
// func TestWaitForAllButNoneStarted(t *testing.T) {
// 	// identical to TestExecutionWithMultipleChildren, b waits for all A to complete
// 	e := NewExecutionPlan()
// 	e.AddDependency(InitialState, "a", false)
// 	e.AddDependency("a", "b", true)

// 	// simulate
// 	e.Completed(InitialState)
// 	next := e.GetPrioritizedNext()
// 	assert.Equal(t, []string{"a"}, next)
// 	// attempt to start "a" but it's not startable, so continue and "b" should fire
// 	next = e.GetNext()
// 	assert.Equal(t, []string{"b"}, next)
// 	e.Started("b")
// 	e.Completed("b")
// 	next = e.GetPrioritizedNext()
// 	assert.Equal(t, next, []string{})
// 	next = e.GetNext()
// 	assert.Equal(t, next, []string{})
// 	assert.True(t, e.Done())
// }

// func TestJoinWithAll(t *testing.T) {
// 	e := NewExecutionPlan()
// 	e.AddDependency(InitialState, "a", false)
// 	e.AddDependency("a", "b", true)
// 	e.AddDependency("a", "c", false)
// 	e.AddDependency("b", "c", true)

// 	// simulate
// 	e.Completed(InitialState)
// 	next := e.GetPrioritizedNext()
// 	assert.Equal(t, []string{"a"}, next)
// 	e.Started("a")
// 	next = e.GetNext()
// 	assert.Equal(t, []string{}, next)

// 	e.Completed("a")
// 	next = e.GetPrioritizedNext()
// 	assert.Equal(t, next, []string{"b"})
// 	e.Started("b")
// 	next = e.GetNext()
// 	assert.Equal(t, next, []string{})

// 	e.Completed("b")
// 	next = e.GetPrioritizedNext()
// 	assert.Equal(t, []string{"c"}, next)
// 	next = e.GetNext()
// 	assert.Equal(t, []string{}, next)
// 	e.Started("c")
// 	e.Completed("c")

// 	assert.True(t, e.Done())
// }
