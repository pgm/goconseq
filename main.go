package goconseq

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/flosch/pongo2"
	"github.com/pgm/goconseq/graph"
	"github.com/pgm/goconseq/model"
	"github.com/pgm/goconseq/parser"
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
		gb.AddRule(rule.Name)
		for _, queryProps := range rule.GetQueryProps() {
			gb.AddRuleConsumes(rule.Name, false, queryProps)
		}
		for _, outputProps := range rule.GetOutputProps() {
			gb.AddRuleProduces(rule.Name, outputProps)
		}
	}
	g := gb.Build()
	// g.Print()
	plan := graph.ConstructExecutionPlan(g)
	return plan
}

type PendingRuleApplication struct {
	name     string
	inputs   *persist.Bindings
	existing *persist.AppliedRule
}

// given a rule that could be applied, determine the rules applications we should create
func GetPendingRuleApplications(db *persist.DB,
	name string,
	query *persist.Query) []PendingRuleApplication {

	pending := make([]PendingRuleApplication, 0)

	// find the inputs that satisfy the query
	var rows []*persist.Bindings
	if query == nil || query.IsEmpty() {
		rows = []*persist.Bindings{persist.EmptyBinding}
	} else {
		rows = persist.ExecuteQuery(db, query)
	}

	for _, inputs := range rows {
		if application := db.FindAppliedRule(name, inputs); application != nil {
			// this has already been run in the current session so ignore it
			log.Printf("This rule application was already executed in the current session. Is this case possible?")
		} else if application := db.FindAppliedRuleInHistory(name, inputs); application != nil {
			// this has already been run in a past session,
			pending = append(pending, PendingRuleApplication{name: name, inputs: inputs, existing: application})
		} else {
			// this has never run, add it to our list of things to run
			pending = append(pending, PendingRuleApplication{name: name, inputs: inputs})
		}
	}

	return pending
}

func localizeArtifact(localizer model.ExecutionBuilder, artifact *persist.Artifact) *persist.Artifact {
	newProps := persist.NewArtifactProperties()
	for k, v := range artifact.Properties.Strings {
		newProps.Strings[k] = v
	}
	for k, fileID := range artifact.Properties.Files {
		localPath, err := localizer.Localize(fileID)
		if err != nil {
			panic(err)
		}
		newProps.Strings[k] = localPath
	}

	return &persist.Artifact{Properties: newProps}
}

func expandTemplate(s string, inputs *persist.Bindings) string {
	// return s
	template := pongo2.Must(pongo2.FromString(s))
	inputsContext := map[string]interface{}{}
	for name, value := range inputs.ByName {
		strings := value.GetArtifacts()[0].Properties.Strings
		inputsContext[name] = strings
	}
	result, err := template.Execute(map[string]interface{}{"inputs": inputsContext})
	if err != nil {
		panic(err)
	}
	return result
}

func renderOutputsAsText(outputs []map[string]string) string {
	j, err := json.Marshal(outputs)
	if err != nil {
		panic(err)
	}

	var sb strings.Builder
	sb.WriteString("cat > results.json <<EOF\n")
	sb.WriteString("{\"outputs\": ")
	sb.Write(j)
	sb.WriteString("}\n")
	sb.WriteString("EOF\n")
	return sb.String()
}

func expandRunStatements(runWith []*model.RunWithStatement, inputs *persist.Bindings, outputs []map[string]string) []*model.RunWithStatement {
	result := make([]*model.RunWithStatement, len(runWith))
	for i, r := range runWith {
		result[i] = &model.RunWithStatement{Executable: expandTemplate(r.Executable, inputs), Script: expandTemplate(r.Script, inputs)}
	}
	if outputs != nil {
		outputsText := renderOutputsAsText(outputs)
		result = append(result, &model.RunWithStatement{Executable: outputsText})
	}
	return result
}

type RunningRuleApplication struct {
	Name string
}

// run query for current rule.
// generate possible rule applications
// if rule application exists, copy it
// if not, run rule. Upon completion, look up each artifact. If existing, attach as output, otherwise create a new one
//

func run(context context.Context, config *model.Config, db *persist.DB) {
	// load rules into memory
	plan := rulesToExecutionPlan(config.Rules)
	listenerUpdates := make(chan *Update)

	// blocking call which waits until a running execution completes
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

	running := make(map[int]*RunningRuleApplication)

	// given a set of rule names to evaluate, run query for each. Returns list of completions of tasks which didn't need to really be run
	processRules := func(next []string) (completions []string, err error) {
		// log.Printf("processRules called with: %v", next)
		completions = make([]string, 0, len(next))

		for _, name := range next {
			query := config.Rules[name].Query
			pendings := GetPendingRuleApplications(db, name, query)
			for _, pending := range pendings {
				if pending.existing == nil {
					appID := db.GetNextApplicationID()

					resumeState := startExec(context, config, appID, pending.name, pending.inputs, listenerUpdates)
					appliedRule, err := db.PersistAppliedRule(appID, pending.name, pending.inputs, resumeState)
					if err != nil {
						return nil, err
					}
					db.AddAppliedRuleToCurrent(appID)

					// update map tracking tasks current running and execution plan
					plan.Started(pending.name)
					running[appliedRule.ID] = &RunningRuleApplication{Name: appliedRule.Name}
				} else {
					db.AddAppliedRuleToCurrent(pending.existing.ID)
					completions = append(completions, pending.name)
				}
			}
		}
		return completions, nil
	}

	completionQueue := []string{graph.InitialState}
	for {
		for len(completionQueue) > 0 {
			nextCompletion := completionQueue[len(completionQueue)-1]
			completionQueue = completionQueue[:len(completionQueue)-1]

			log.Printf("completed: %s", nextCompletion)
			plan.Completed(nextCompletion)

			next := plan.GetPrioritizedNext()
			nextCompletions, err := processRules(next)
			if err != nil {
				panic(err)
			}
			completionQueue = append(completionQueue, nextCompletions...)

			next = plan.GetNext()
			nextCompletions, err = processRules(next)
			if err != nil {
				panic(err)
			}
			completionQueue = append(completionQueue, nextCompletions...)
		}

		// log.Printf("plan.Done() = %v running = %v", plan.Done(), running)
		if plan.Done() && len(running) == 0 {
			break
		}

		ruleApplicationID, completionState := getNextCompletion()
		// log.Printf("getNextCompletion returned ruleApplicationID=%v, model.CompletionState=%v", ruleApplicationID, completionState)
		success := completionState.Success
		delete(running, ruleApplicationID)

		var failureMessage string
		var outputs []*persist.ArtifactProperties

		if success {
			// attempt to parse the results
			var err error
			outputs, err = readResultOutputs(db.GetWorkDir(ruleApplicationID), func(filename string) (int, error) {
				panic("unimp")
			})
			if err != nil {
				success = false
				failureMessage = err.Error()
			}
		} else {
			failureMessage = completionState.FailureMessage
		}

		if success {
			// write all of the artifacts to the DB
			outputArtifacts := make([]*persist.Artifact, len(outputs))
			for i, props := range outputs {
				artifact := db.FindArtifactInHistory(props)
				if artifact == nil {
					artifact, err := db.PersistArtifact(props)
					if err != nil {
						panic(err)
					}
				}
				outputArtifacts[i] = artifact
			}

			// mark applied rule as complete
			db.UpdateAppliedRuleComplete(ruleApplicationID, outputArtifacts)

			// notify the scheduler that this rule completed
			appliedRule := db.GetAppliedRule(ruleApplicationID)
			completionQueue = append(completionQueue, appliedRule.Name)
		} else {
			log.Printf("Error: %s", failureMessage)

			err := db.DeleteAppliedRule(ruleApplicationID)
			if err != nil {
				panic(err)
			}
		}
	}
}

func startExec(context context.Context, config *model.Config, id int, name string, inputs *persist.Bindings, listenerUpdates chan *Update) string {
	listener := &execListener{ruleApplicationID: id, c: listenerUpdates}
	rule := config.Rules[name]
	executorName := rule.ExecutorName
	executor := config.Executors[executorName]
	builder := executor.Builder(id)
	localizedInputs := inputs.Transform(func(artifact *persist.Artifact) *persist.Artifact {
		return localizeArtifact(builder, artifact)
	})
	runStatements := expandRunStatements(rule.RunStatements, localizedInputs, rule.Outputs)
	builder.Prepare(runStatements)

	process, err := builder.Start(context)
	if err != nil {
		panic(err)
	}

	resumeState := process.GetResumeState()
	go process.Wait(listener)

	return resumeState
}

func readResultOutputs(workDir string, getFileID func(filename string) (int, error)) (Properties []*persist.ArtifactProperties, err error) {
	data, err := readJson(path.Join(workDir, "results.json"), getFileID)
	if err != nil {
		return nil, err
	}

	// todo, add checks for each of these
	m := data.(map[string]interface{})
	outputs := m["outputs"].([]interface{})
	artifacts := make([]*persist.ArtifactProperties, len(outputs))
	for i, output := range outputs {
		artifacts[i] = artifactPropsFromJson(output, getFileID)
	}

	return artifacts, nil
}

func readJson(filename string, getFileID func(filename string) (int, error)) (interface{}, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var data interface{}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func artifactPropsFromJson(json interface{}, getFileID func(filename string) (int, error)) *persist.ArtifactProperties {
	jsonObj := json.(map[string]interface{})
	artifact := persist.NewArtifactProperties()
	for key, value := range jsonObj {
		valueMap, ok := value.(map[string]interface{})
		if ok {
			filename := valueMap["$filename"].(string)
			fileID, err := getFileID(filename)
			if err != nil {
				// todo: gracefully handle errors
				panic(err)
			}
			// todo: check for dup key
			artifact.Files[key] = fileID
		} else {
			// todo: check for dup key
			artifact.Strings[key] = value.(string)
		}
	}
	return artifact
}

func parseFile(config *model.Config, filename string) error {
	statements, err := parser.ParseFile(filename)
	if err != nil {
		return err
	}
	statements.Eval(config)
	return nil
}

func RunRulesInFile(stateDir string, filename string) error {
	config := model.NewConfig()
	config.StateDir = stateDir

	db := persist.NewDB(stateDir)
	config.Executors[model.DefaultExecutorName] = &LocalExec{jobDir: stateDir}

	err := parseFile(config, filename)
	if err != nil {
		return err
	}

	run(context.Background(), config, db)
	return nil
}
