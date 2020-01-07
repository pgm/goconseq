package goconseq

import (
	"context"
	"log"

	"github.com/pgm/goconseq/graph"
	"github.com/pgm/goconseq/model"
	"github.com/pgm/goconseq/persist"
)

type Update struct {
	CompletionState   *model.CompletionState
	status            *string
	ruleApplicationID int
}

type execListener struct {
	ruleApplicationID int
	c                 chan *Update
}

func (e *execListener) Completed(state *model.CompletionState) {
	e.c <- &Update{ruleApplicationID: e.ruleApplicationID, CompletionState: state}
}

func (e *execListener) UpdateStatus(status string) {
	e.c <- &Update{ruleApplicationID: e.ruleApplicationID, status: &status}
}

func rulesToExecutionPlan(rules map[string]*model.Rule) *graph.ExecutionPlan {
	gb := graph.NewGraphBuilder()
	for _, rule := range rules {
		for _, queryProps := range rule.GetQueryProps() {
			gb.AddRuleConsumes(rule.Name, false, queryProps)
		}
		for _, outputProps := range rule.GetOutputProps() {
			gb.AddRuleProduces(rule.Name, outputProps)
		}
	}
	g := gb.Build()
	g.Print()
	plan := graph.ConstructExecutionPlan(g)
	return plan
}

func ProcessRule(db *persist.DB, name string, query *persist.Query, startCallback func(id int, name string, inputs *persist.Bindings) string) (int, error) {
	started := 0
	var rows []*persist.Bindings
	if query == nil {
		rows = []*persist.Bindings{persist.EmptyBinding}
	} else {
		rows = persist.ExecuteQuery(db, query)
	}
	for _, inputs := range rows {
		// does this application already exist?
		application := db.FindAppliedRule(name, inputs)
		if application != nil {
			// if it exists, nothing to do
			continue
		}

		applicationID := db.GetNextApplicationID()
		resumeState := startCallback(applicationID, name, inputs)
		_, err := db.PersistAppliedRule(applicationID, name, inputs, resumeState)
		if err != nil {
			return 0, err
		}
		started++
	}

	return started, nil
}

func generateCommand(rule *model.Rule, inputs *persist.Bindings) string {
	if rule.RunStatements != nil {
		panic("unimplemented")
	}
	return "date"
}

func localizeArtifact(localizer model.Localizer, artifact *persist.Artifact) *persist.Artifact {
	var newArtifact persist.Artifact
	for k, v := range artifact.Properties.Strings {
		newArtifact.Properties.Strings[k] = v
	}
	for k, fileID := range artifact.Properties.Files {
		localPath, err := localizer.Localize(fileID)
		if err != nil {
			panic(err)
		}
		newArtifact.Properties.Strings[k] = localPath
	}
	return &newArtifact
}

func run(context context.Context, config *model.Config) *persist.DB {
	// load rules into memory
	db := persist.NewDB()
	plan := rulesToExecutionPlan(config.Rules)
	listenerUpdates := make(chan *Update)

	getNextCompletion := func() (int, *model.CompletionState) {
		for {
			update := <-listenerUpdates

			if update.CompletionState != nil {
				log.Printf("ID: %d model.CompletionState: %v", update.ruleApplicationID, update.CompletionState)
				return update.ruleApplicationID, update.CompletionState
			}
			if update.status != nil {
				log.Printf("ID: %d status: %s", update.ruleApplicationID, *update.status)
			}
		}
	}

	startCallback := func(id int, name string, inputs *persist.Bindings) string {
		listener := &execListener{ruleApplicationID: id, c: listenerUpdates}
		// execution := ExecutionFactory.Create()
		// command := generateCommand(execution, name, inputs)
		rule := config.Rules[name]
		executorName := rule.ExecutorName
		executor := config.Executors[executorName]
		localizer := executor.GetLocalizer()
		localizedInputs := inputs.Transform(func(artifact *persist.Artifact) *persist.Artifact {
			return localizeArtifact(localizer, artifact)
		})
		command := generateCommand(rule, localizedInputs)

		// special case: nothing to run for this rule. primarily used by tests
		if command == "" {
			plan.Started(name)
			listener.Completed(&model.CompletionState{Success: true})
			return ""
		}
		process, err := executor.Start(context, []string{command}, localizer)
		if err != nil {
			panic(err)
		}

		plan.Started(name)

		resumeState := process.GetResumeState()
		go process.Wait(listener)

		return resumeState
	}

	running := 1

	processRules := func(next []string) error {
		log.Printf("processRules called with: %v", next)
		for _, name := range next {
			query := config.Rules[name].Query
			started, err := ProcessRule(db, name, query, startCallback)
			if err != nil {
				return err
			}
			running += started
		}
		return nil
	}

	nextCompletion := graph.InitialState
	for {
		log.Printf("completed: %s", nextCompletion)
		plan.Completed(nextCompletion)
		running--
		next := plan.GetPrioritizedNext()
		processRules(next)
		next = plan.GetNext()
		processRules(next)

		if plan.Done() && running == 0 {
			break
		}

		for {
			ruleApplicationID, CompletionState := getNextCompletion()
			log.Printf("getNextCompletion returned ruleApplicationID=%v, model.CompletionState=%v", ruleApplicationID, CompletionState)
			if CompletionState.Success {
				appliedRule := db.GetAppliedRule(ruleApplicationID)
				nextCompletion = appliedRule.Name
				break
			} else {
				err := db.DeleteAppliedRule(ruleApplicationID)
				if err != nil {
					panic(err)
				}
			}
		}
	}

	return db
}
