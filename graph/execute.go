package graph

import "sort"

const InitialState = "<INITIAL_STATE>"

type ExecutionPlan struct {
	afterEach map[string]*strSet
	blockedBy map[string]*strSet

	runningRules map[string]int

	pendingRules strSet
}

func NewExecutionPlan() *ExecutionPlan {
	e := &ExecutionPlan{
		afterEach:    make(map[string]*strSet),
		blockedBy:    make(map[string]*strSet),
		runningRules: make(map[string]int)}

	e.runningRules[InitialState] = 1

	return e
}

func (e *ExecutionPlan) Done() bool {
	return e.pendingRules.Length() == 0
}

func (e *ExecutionPlan) AddDependency(precursor string, successor string) bool {
	// if len(e.blockedBy) > 0 {
	// 	panic("Cannot call AddDependency after AddBlockedBy is called")
	// }
	return e.updateAfterEach(precursor, successor)
}

func (e *ExecutionPlan) updateAfterEach(precursor string, successor string) bool {
	s := e.afterEach[precursor]
	if s == nil {
		s = &strSet{}
		e.afterEach[precursor] = s
	}
	return s.Add(successor)
}

func (e *ExecutionPlan) AddBlockedBy(precursor string, successor string) {
	e.updateBlockedBy(precursor, successor)

	// // invert the afterEach index so we can look up by successors
	// bySuccessor := make(map[string]*strSet)
	// for precursor, ss := range e.afterEach {
	// 	ss.ForEach(func(successor string) {
	// 		p := bySuccessor[successor]
	// 		if p == nil {
	// 			p = &strSet{}
	// 			bySuccessor[successor] = p
	// 		}
	// 		p.Add(precursor)
	// 	})
	// }

	// // for each transitive precursor, we want to update blocking list
	// // this isn't quite right because transitiveUpdateBlockedBy iterates through e.afterEach _and_ mutates e.afterEach
	// e.transitiveUpdateBlockedBy(precursor, successor, bySuccessor)
}

func (e *ExecutionPlan) updateBlockedBy(precursor string, successor string) {
	s := e.blockedBy[successor]
	if s == nil {
		s = &strSet{}
		e.blockedBy[successor] = s
	}
	s.Add(precursor)
}

// func (e *ExecutionPlan) transitiveUpdateBlockedBy(successor string, terminalRule string, bySuccessor map[string]*strSet) {
// 	ss := bySuccessor[successor]
// 	if ss != nil {
// 		ss.ForEach(func(precursor string) {
// 			e.updateBlockedBy(precursor, terminalRule)
// 			e.transitiveUpdateBlockedBy(precursor, terminalRule, bySuccessor)
// 		})
// 	}
// }

func (e *ExecutionPlan) Completed(name string) {
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
		blockedBy := e.blockedBy[name]
		if blockedBy != nil {
			blockedBy.ForEach(func(blocker string) {
				if checkPending {
					// if this precursor is pending, don't add
					if e.pendingRules.Has(blocker) {
						addToNext = false
					}
				}
				// if the precursor is running, don't add
				if e.runningRules[blocker] > 0 {
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

type transitiveApplyRec struct {
	// transitivly add relationship(a, b) and then apply relationship(x, b) for all x in a's precursors
	a string
	b string
}

type strToStrSet struct {
	m map[string]*strSet
}

func newStrToStrSet() *strToStrSet {
	return &strToStrSet{make(map[string]*strSet)}
}

func (m *strToStrSet) Add(key string, value string) bool {
	s := m.m[key]
	if s == nil {
		s = newStrSet()
		m.m[key] = s
	}
	return s.Add(value)
}

func (m *strToStrSet) Get(key string) []string {
	s := m.m[key]
	if s == nil {
		return nil
	}
	result := make([]string, 0, s.Length())
	s.ForEach(func(v string) {
		result = append(result, v)
	})
	return result
}

func ConstructExecutionPlan(g *Graph) *ExecutionPlan {
	plan := NewExecutionPlan()
	initialAllRules := make([]transitiveApplyRec, 0, 100)

	// map of rule -> prior rule names
	priorRules := newStrToStrSet()

	// add all direct dependencies
	g.ForEachRule(func(r *Rule) {
		// precursor string, successor string, waitForAll bool
		if len(r.consumes) == 0 {
			plan.AddDependency(InitialState, r.name)
			priorRules.Add(r.name, InitialState)
		} else {
			for _, a := range r.consumes {
				for _, precursor := range a.artifact.producedBy {
					plan.AddDependency(precursor.name, r.name)
					priorRules.Add(r.name, precursor.name)
					if a.isAll {
						initialAllRules = append(initialAllRules, transitiveApplyRec{precursor.name, r.name})
					}
				}
			}
		}
	})

	addedAtLeastOne := true
	for addedAtLeastOne {
		addedAtLeastOne = false

		pendingAdd := make([]transitiveApplyRec, len(initialAllRules))
		copy(pendingAdd, initialAllRules)

		for len(pendingAdd) > 0 {
			// take next element
			p := pendingAdd[len(pendingAdd)-1]
			pendingAdd = pendingAdd[:len(pendingAdd)-1]

			plan.AddDependency(p.a, p.b)
			plan.AddBlockedBy(p.a, p.b)
			newlyAdded := priorRules.Add(p.b, p.a)

			if newlyAdded {
				addedAtLeastOne = true
			}
			for _, nextPrior := range priorRules.Get(p.a) {
				pendingAdd = append(pendingAdd, transitiveApplyRec{nextPrior, p.b})
			}
		}
	}

	return plan
}

// func testLinearGraph() {
// 	gb := NewGraphBuilder()
// 	gb.AddRuleProduces("r1", parseProps("p:a1"))
// 	gb.AddRuleConsumes("r2", false, parseProps("p:a1"))
// 	gb.AddRuleProduces("r2", parseProps("p:a2"))
// 	gb.AddRuleConsumes("r3", false, parseProps("p:a3"))
// 	g := gb.Build()
// 	assert(len(g.roots) == 1)
// }

// func testForkJoinGraph() {
// 	gb := NewGraphBuilder()
// 	gb.AddRuleProduces("r1", parseProps("p1:a1", "p2:a1"))
// 	gb.AddRuleConsumes("r2", false, parseProps("p1:a1"))
// 	gb.AddRuleProduces("r2", parseProps("p2:a2"))
// 	gb.AddRuleConsumes("r3", false, parseProps("p2:a1"))
// 	gb.AddRuleProduces("r3", parseProps("p2:a2"))
// 	gb.AddRuleConsumes("r4", false, parseProps("p2:a2"))
// 	g := gb.Build()
// 	assert(len(g.roots) == 1)
// }

//////
