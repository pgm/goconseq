package goconseq

import (
	"context"
	"log"

	"./persist"
)

type RunWithStatement struct {
	Script     string
	Executable string
}

type Rule struct {
	Name              string
	Query             *persist.Query
	ExecutorName      string
	RequiredResources map[string]float64
	RunStatements     []*RunWithStatement
}

type Config struct {
	Rules     map[string]*Rule
	Vars      map[string]string
	Artifacts []propPairs
	Executors map[string]map[string]string
}

type Update struct {
	resumeState       *string
	completionState   *CompletionState
	status            *string
	ruleApplicationID int
}

type execListener struct {
	ruleApplicationID int
	c                 chan *Update
}

func (e *execListener) Started(resumeState string) {
	e.c <- &Update{ruleApplicationID: e.ruleApplicationID, resumeState: &resumeState}
}

func (e *execListener) Completed(state *CompletionState) {
	e.c <- &Update{ruleApplicationID: e.ruleApplicationID, completionState: state}
}

func (e *execListener) UpdateStatus(status string) {
	e.c <- &Update{ruleApplicationID: e.ruleApplicationID, status: &status}
}

func main(context context.Context) {
	// load rules into memory
	gb := NewGraphBuilder()
	g := gb.Build()
	db := persist.NewDB()
	plan := NewExecutionPlan(g)

	listenerUpdates := make(chan *Update)

	getNextCompletion := func() (int, *CompletionState) {
		for {
			update := <-listenerUpdates
			if update.resumeState != nil {
				log.Printf("ID: %d resumeState: %s", update.ruleApplicationID, update.resumeState)
			}
			if update.completionState != nil {
				log.Printf("ID: %d completionState: %v", update.ruleApplicationID, update.completionState)
				return update.ruleApplicationID, update.completionState
			}
			if update.status != nil {
				log.Printf("ID: %d status: %s", update.ruleApplicationID, update.status)
			}
		}
	}

	startCallback := func(id int, name string, inputs *Bindings) string {
		listener := &execListener{ruleApplicationID: id, c: listenerUpdates}
		execution := ExecutionFactory.Create()
		command := generateCommand(execution, name, inputs)
		execution.Start(context, command, listener)
		return resumeState
	}

	running := 0

	processRules := func(next []string) error {
		for _, name := range next {
			started, err := ProcessRule(db, name, queryByName[name], startCallBack)
			if err != nil {
				return err
			}
			running += started
		}
		return nil
	}

	nextCompletion := InitialState
	for {
		plan.Completed(nextCompletion)
		next := plan.GetPrioritizedNext()
		processRules(next)
		next = plan.GetNext()
		processRules(next)

		if plan.Done() && running == 0 {
			break
		}

		ruleApplicationID, completionState := getNextCompletion()
	}
	// complete(Initial)
	// do mainLoop
}
