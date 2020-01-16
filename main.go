package goconseq

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

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
		gb.AddRule(rule.Name)
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

func ProcessRule(db *persist.DB,
	name string,
	query *persist.Query,
	startCallback func(id int, name string, inputs *persist.Bindings) string) (int, error) {
	started := 0
	var rows []*persist.Bindings
	if query == nil || query.IsEmpty() {
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

	return &persist.Artifact{ProducedBy: -1, Properties: newProps}
}

func expandTemplate(s string, vars *persist.Bindings) string {
	if strings.Contains(s, "{{") {
		panic("templates not implemented")
	}
	return s
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

func run(context context.Context, config *model.Config, db *persist.DB) {
	// load rules into memory
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

	running := make(map[int]*RunningRuleApplication)

	startCallback := func(id int, name string, inputs *persist.Bindings) string {
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

		// // special case: nothing to run for this rule. primarily used by tests
		// if command == "" {
		// 	plan.Started(name)
		// 	listener.Completed(&model.CompletionState{Success: true})
		// 	return ""
		// }
		process, err := builder.Start(context)
		if err != nil {
			panic(err)
		}

		plan.Started(name)

		running[id] = &RunningRuleApplication{Name: name}

		resumeState := process.GetResumeState()
		go process.Wait(listener)

		return resumeState
	}

	processRules := func(next []string) error {
		log.Printf("processRules called with: %v", next)
		for _, name := range next {
			query := config.Rules[name].Query
			_, err := ProcessRule(db, name, query, startCallback)
			if err != nil {
				return err
			}
		}
		return nil
	}

	nextCompletion := graph.InitialState
	for {
		if nextCompletion != "" {
			log.Printf("completed: %s", nextCompletion)
			plan.Completed(nextCompletion)
			next := plan.GetPrioritizedNext()
			processRules(next)
			next = plan.GetNext()
			processRules(next)
		}

		log.Printf("plan.Done() = %v running = %v", plan.Done(), running)
		if plan.Done() && len(running) == 0 {
			break
		}

		ruleApplicationID, completionState := getNextCompletion()
		log.Printf("getNextCompletion returned ruleApplicationID=%v, model.CompletionState=%v", ruleApplicationID, completionState)
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
				artifact, err := db.PersistArtifact(ruleApplicationID, props)
				if err != nil {
					panic(err)
				}
				outputArtifacts[i] = artifact
			}

			// mark applied rule as complete
			db.UpdateAppliedRuleComplete(ruleApplicationID, outputArtifacts)

			// notify the scheduler that this rule completed
			appliedRule := db.GetAppliedRule(ruleApplicationID)
			nextCompletion = appliedRule.Name
		} else {
			log.Printf("Error: %s", failureMessage)

			err := db.DeleteAppliedRule(ruleApplicationID)
			if err != nil {
				panic(err)
			}
			nextCompletion = ""
		}
	}
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
