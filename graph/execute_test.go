package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleExecution(t *testing.T) {
	e := NewExecutionPlan()
	e.AddDependency(InitialState, "a")
	e.AddDependency("a", "b")

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
	e.AddDependency(InitialState, "a")
	e.AddDependency("a", "b")

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
	e.AddDependency(InitialState, "a")
	e.AddDependency("a", "b")

	// simulate
	e.Completed(InitialState)
	next := e.GetPrioritizedNext()
	assert.Equal(t, []string{"a"}, next)
	next = e.GetNext()
	assert.Equal(t, []string{}, next)

	assert.True(t, e.Done())
}

func TestWaitForAll(t *testing.T) {
	// identical to TestExecutionWithMultipleChildren, b waits for all A to complete
	e := NewExecutionPlan()
	e.AddDependency(InitialState, "a")
	e.AddDependency("a", "b")
	e.AddDependency(InitialState, "b")

	e.AddBlockedBy("a", "b")

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
	assert.Equal(t, next, []string{})
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
	assert.Equal(t, []string{}, next)
	next = e.GetNext()
	assert.Equal(t, []string{}, next)

	assert.True(t, e.Done())
}

// this test fails because b would need to be added to the pending list after InitialState
func TestWaitForAllButNoneStarted(t *testing.T) {
	// identical to TestExecutionWithMultipleChildren, b waits for all A to complete
	e := NewExecutionPlan()
	e.AddDependency(InitialState, "a")
	e.AddDependency("a", "b")
	e.AddDependency(InitialState, "b")
	e.AddBlockedBy("a", "b")
	e.AddBlockedBy(InitialState, "b")

	// peek inside and verify that AddBlockedBy updated InitialState's successor list
	assert.Equal(t, "{a b}", e.afterEach[InitialState].String())
	assert.Equal(t, "{"+InitialState+" a}", e.blockedBy["b"].String())

	// simulate
	e.Completed(InitialState)
	// peek inside the internals at the pending list
	assert.Equal(t, "{a b}", e.pendingRules.String())
	next := e.GetPrioritizedNext()
	assert.Equal(t, []string{"a"}, next)
	// now "b" should be pending because it should have been added as a dependency of initialState when AddBlockedBy was called
	assert.Equal(t, "{b}", e.pendingRules.String())
	// attempt to start "a" but it's not startable, so continue and "b" should fire
	next = e.GetNext()
	assert.Equal(t, []string{"b"}, next)
	e.Started("b")
	e.Completed("b")
	next = e.GetPrioritizedNext()
	assert.Equal(t, next, []string{})
	next = e.GetNext()
	assert.Equal(t, next, []string{})
	assert.True(t, e.Done())
}

func TestJoinWithAll(t *testing.T) {
	e := NewExecutionPlan()
	// initial -> a
	// a -> c
	// all a -> b
	// all b -> c
	e.AddDependency(InitialState, "a")
	e.AddDependency("a", "c")
	// implied deps
	e.AddDependency(InitialState, "b")
	e.AddDependency(InitialState, "c")
	e.AddDependency("a", "b")
	e.AddDependency("a", "c")
	e.AddBlockedBy("a", "b")
	e.AddBlockedBy(InitialState, "b")
	e.AddBlockedBy("b", "c")
	e.AddBlockedBy("a", "c")
	e.AddBlockedBy(InitialState, "c")

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
	assert.Equal(t, []string{"c"}, next)
	next = e.GetNext()
	assert.Equal(t, []string{}, next)
	e.Started("c")
	e.Completed("c")

	assert.True(t, e.Done())
}
