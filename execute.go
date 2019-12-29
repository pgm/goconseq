package goconseq

import "sort"

const InitialState = "<INITIAL_STATE>"

type strSet struct {
	set map[string]bool
}

func (ss *strSet) Add(value string) {
	if ss.set == nil {
		ss.set = make(map[string]bool)
	}
	ss.set[value] = true
}

func (ss *strSet) ForEach(fn func(string)) {
	if ss.set == nil {
		return
	}
	for value := range ss.set {
		fn(value)
	}
}

func (ss *strSet) Length() int {
	if ss.set == nil {
		return 0
	}
	return len(ss.set)
}

func (ss *strSet) Remove(value string) {
	if ss.set == nil {
		return
	}
	delete(ss.set, value)
}

func (ss *strSet) Has(value string) bool {
	if ss.set == nil {
		return false
	}
	return ss.set[value]
}

type ExecutionPlan struct {
	afterEach map[string]*strSet
	afterAll  map[string]*strSet

	runningRules map[string]int

	pendingRules strSet

	fullyInitialized bool
}

func NewExecutionPlan() *ExecutionPlan {
	e := &ExecutionPlan{
		afterEach:    make(map[string]*strSet),
		afterAll:     make(map[string]*strSet),
		runningRules: make(map[string]int)}

	e.runningRules[InitialState] = 1

	return e
}

func (e *ExecutionPlan) Done() bool {
	return e.pendingRules.Length() == 0
}

func (e *ExecutionPlan) AddDependency(precursor string, successor string, waitForAll bool) {
	s := e.afterEach[precursor]
	if s == nil {
		s = &strSet{}
		e.afterEach[precursor] = s
	}
	s.Add(successor)

	if waitForAll {
		s = e.afterAll[successor]
		if s == nil {
			s = &strSet{}
			e.afterAll[successor] = s
		}
		s.Add(precursor)
	}
}

func (e *ExecutionPlan) Completed(name string) {
	if !e.fullyInitialized {
		// TODO: this logic and all transient relationships into construction of execplan

		// add AfterAll relationships transiently based on forEach relationships
		for successor, precursors := range e.afterAll {
			precursors.ForEach(func(precursor string) {
				e.AddDependency(precursor, successor, false)
			})
		}

		e.fullyInitialized = true
	}

	afterEach := e.afterEach[name]
	if afterEach != nil {
		afterEach.ForEach(func(name string) {
			e.pendingRules.Add(name)
		})
	}
	e.runningRules[name]--
	if e.runningRules[name] < 0 {
		panic("completed a rule not yet running")
	}
}

func (e *ExecutionPlan) Started(name string) {
	e.runningRules[name]++
}

func (e *ExecutionPlan) getNext(checkPending bool) []string {
	// first do any rules which do not need to wait for any currently running rule _or_ other pending rule
	next := make([]string, 0, e.pendingRules.Length())
	e.pendingRules.ForEach(func(name string) {
		addToNext := true

		// find all the rules that we need to wait for full completion before starting
		afterAll := e.afterAll[name]
		if afterAll != nil {
			afterAll.ForEach(func(fullyWaitPrecursor string) {
				if checkPending {
					// if this precursor is pending, don't add
					if e.pendingRules.Has(fullyWaitPrecursor) {
						addToNext = false
					}
				}
				// if the precursor is running, don't add
				if e.runningRules[fullyWaitPrecursor] > 0 {
					addToNext = false
				}
			})
		}
		if addToNext {
			next = append(next, name)
		}
	})

	// sort the rule names in next to ensure deterministic ordering
	sort.Strings(next)

	// before returning next, remove these from the pending list
	for _, name := range next {
		e.pendingRules.Remove(name)
	}
	return next
}

func (e *ExecutionPlan) GetPrioritizedNext() []string {
	return e.getNext(true)
}

func (e *ExecutionPlan) GetNext() []string {
	return e.getNext(false)
}

// invoke in the following order
// names = ExecutionPlan.GetPrioritizedNext()
// ...ExecutionPlan.Started(name)
// names = ExecutionPlan.GetNext()
// ...ExecutionPlan.Started(name)
// ...ExecutionPlan.Completed(completed_name)
