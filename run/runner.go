package run

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/flosch/pongo2"
	"github.com/pgm/goconseq/executor"
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

func rulesToGraph(rules map[string]*model.Rule) *graph.Graph {
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
	return gb.Build()
}

type PendingRuleApplication struct {
	name     string
	hash     string
	inputs   *persist.Bindings
	existing *persist.AppliedRule
}

// given a rule that could be applied, determine the rules applications we should create
func GetPendingRuleApplications(db *persist.DB,
	name string,
	hash string,
	query model.QueryI,
	replayOnly bool) []PendingRuleApplication {

	pending := make([]PendingRuleApplication, 0)

	// find the inputs that satisfy the query
	var rows []*persist.Bindings
	if query == nil || query.IsEmpty() {
		rows = []*persist.Bindings{persist.EmptyBinding}
	} else {
		r1 := query.ExecuteQuery(db)
		log.Printf("Executing query for %s: %v returned %d rows", name, query.AsDict(), len(r1))
		db.DumpArtifacts()
		rows = make([]*persist.Bindings, len(r1))
		for i, r1v := range r1 {
			rows[i] = r1v.(*persist.Bindings)
		}
	}

	for _, inputs := range rows {
		log.Printf("inputs=%v, currentAppliedRules=%d", inputs, db.GetHackCount())
		if application := db.FindAppliedRule(name, hash, inputs); application != nil {
			// this has already been run in the current session so ignore it
			log.Printf("This rule application was already executed in the current session. Is this case possible?")
		} else if application := db.GetAppliedRuleFromHistory(name, hash, inputs); application != nil {
			// this has already been run in a past session,
			// log.Printf("Found in existing session")
			pending = append(pending, PendingRuleApplication{name: name, hash: hash, inputs: inputs, existing: application})
		} else if !replayOnly {
			// this has never run, add it to our list of things to run
			//			log.Printf("never run and replayOnly = %v", replayOnly)
			pending = append(pending, PendingRuleApplication{name: name, hash: hash, inputs: inputs})
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
		_, ok := value.(*persist.SingleArtifact)
		if ok {
			strings := value.GetArtifacts()[0].Properties.Strings
			inputsContext[name] = strings
		} else {
			log.Printf("TODO: expandTemplate does not handle multiple artifact variables")
		}
	}
	result, err := template.Execute(map[string]interface{}{"inputs": inputsContext})
	if err != nil {
		panic(err)
	}
	return result
}

func transformMapValues(orig map[string]string, transform func(string) string) map[string]string {
	m := make(map[string]string)
	for k, value := range orig {
		m[k] = transform(value)
	}
	return m
}

func transformRuleOutput(orig *model.RuleOutput, fileLookup func(int) string, transform func(string) string) map[string]interface{} {
	m := make(map[string]interface{})
	for _, prop := range orig.Properties {
		if prop.IsFilename {
			fn := transform(prop.Value)
			m[prop.Name] = map[string]string{"$filename": fn}
		} else {
			m[prop.Name] = transform(prop.Value)
		}
	}
	return m
}

func renderOutputsAsText(builder model.ExecutionBuilder, outputs []map[string]interface{}) string {
	results := make(map[string]interface{})
	results["outputs"] = outputs
	j, err := json.Marshal(results)
	if err != nil {
		panic(err)
	}
	outputsAsJsonPath, err := builder.AddFile(j)
	if err != nil {
		// todo handle gracefully
		panic(err)
	}

	var sb strings.Builder
	sb.WriteString("cp ")
	sb.WriteString(outputsAsJsonPath)
	sb.WriteString(" results.json")
	// > results.json <<EOF\n")
	// sb.WriteString("{\"outputs\": ")
	// sb.Write(j)
	// sb.WriteString("}\n")
	// sb.WriteString("EOF\n")
	return sb.String()
}

func expandRunStatements(runWith []*model.RunWithStatement, inputs *persist.Bindings, outputs []model.RuleOutput,
	localPathLookup func(fileID int) string, builder model.ExecutionBuilder) []*model.RunWithStatement {
	result := make([]*model.RunWithStatement, len(runWith))
	for i, r := range runWith {
		result[i] = &model.RunWithStatement{Executable: expandTemplate(r.Executable, inputs), Script: expandTemplate(r.Script, inputs)}
	}
	if outputs != nil {
		expandedOutputs := make([]map[string]interface{}, len(outputs))
		for i, output := range outputs {
			expandedOutputs[i] = transformRuleOutput(&output,
				localPathLookup,
				func(x string) string {
					return expandTemplate(x, inputs)
				})
		}
		outputsText := renderOutputsAsText(builder, expandedOutputs)
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

type RunStats struct {
	RuleEvaluations       int
	ExistingAppliedRules  int
	Executions            int
	SuccessfulCompletions int
	FailedCompletions     int
}

func computeSha256(filename string) (string, error) {
	hasher := sha256.New()
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func AddArtifactRule(c *model.Config, fileRepo model.FileRepository) {
	outputs := make([]model.RuleOutput, 0, len(c.Artifacts))
	log.Printf("Warning: need to change AddArtifactRule to create one rule per artifact")

	for _, artifact := range c.Artifacts {
		output := model.RuleOutput{Properties: make([]model.RuleOutputProperty, 0, len(artifact))}

		for key, value := range artifact {
			if value.IsFilename {
				filename := value.Value
				sha256, err := computeSha256(filename)
				if err != nil {
					log.Panicf("Could not read %s: %s", filename, err)
				}

				// fileID := fileRepo.AddFileOrFind(filename, sha256)

				// // output.AddPropertyString("type", model.FileRefType)
				output.AddPropertyString(key+"$sha256", sha256)
				// // output.AddPropertyString("filename", filename)
				output.AddPropertyFilename(key, value.Value)
			} else {
				output.AddPropertyString(key, value.Value)
			}
		}

		outputs = append(outputs, output)
	}

	rule := &model.Rule{Name: "<artifact rule>",
		Outputs:      outputs,
		ExecutorName: model.DefaultExecutorName}

	c.AddRule(rule)
}

func runAndGetGraph(context context.Context, config *model.Config, db *persist.DB) (*graph.Graph, *RunStats) {
	// make a synthetic rule which emits all the artifacts in the config
	if len(config.Artifacts) > 0 {
		AddArtifactRule(config, db)
	}

	// load rules into memory
	execGraph := rulesToGraph(config.Rules)

	stats := innerRun(context, config, execGraph, db)

	return execGraph, stats
}

func innerRun(context context.Context, config *model.Config, execGraph *graph.Graph, db *persist.DB) *RunStats {
	var stats RunStats

	localPathLookup := func(fileID int) string {
		return db.GetFile(fileID).LocalPath
	}

	plan := graph.ConstructExecutionPlan(execGraph)
	listenerUpdates := make(chan *Update, 100)

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
			stats.RuleEvaluations++

			rule := config.Rules[name]
			query := rule.Query
			hash := rule.Hash()
			log.Printf("rule %s hash: %s", name, hash)
			pendings := GetPendingRuleApplications(db, name, hash, query, config.ReplayOnly)

			for _, pending := range pendings {
				if pending.existing == nil {
					stats.Executions++

					appID := db.GetNextApplicationID()

					resumeState := startExec(context, config, localPathLookup, appID, pending.name, pending.inputs, listenerUpdates)
					appliedRule, err := db.PersistAppliedRule(appID, pending.name, pending.hash, pending.inputs, resumeState)
					if err != nil {
						return nil, err
					}
					db.AddAppliedRuleToCurrent(appID)

					// update map tracking tasks current running and execution plan
					plan.Started(pending.name)
					running[appliedRule.ID] = &RunningRuleApplication{Name: appliedRule.Name}
				} else {
					stats.ExistingAppliedRules++

					db.AddAppliedRuleToCurrent(pending.existing.ID)
					plan.Started(pending.name)
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
				sha256, err := computeSha256(filename)
				if err != nil {
					return 0, fmt.Errorf("Could not read %s: %s", filename, err)
				}

				fileID := db.AddFileOrFind(filename, sha256)
				return fileID, nil
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
			log.Printf("Completed %s", running[ruleApplicationID])
			for i, props := range outputs {
				log.Printf("output artifact %d: %s", i, props.String())
				artifact := db.GetArtifactFromHistory(props)
				if artifact == nil {
					var err error
					artifact, err = db.PersistArtifact(props)
					if err != nil {
						panic(err)
					}
				}
				outputArtifacts[i] = artifact
			}

			// mark applied rule as complete
			err := db.UpdateAppliedRuleComplete(ruleApplicationID, outputArtifacts)
			if err != nil {
				failureMessage = err.Error()
				success = false
			} else {
				// notify the scheduler that this rule completed
				stats.SuccessfulCompletions++
				appliedRule := db.GetAppliedRule(ruleApplicationID)
				completionQueue = append(completionQueue, appliedRule.Name)
			}
		}

		if !success {
			stats.FailedCompletions++

			log.Printf("Error: %s", failureMessage)

			err := db.DeleteAppliedRule(ruleApplicationID)
			if err != nil {
				panic(err)
			}
		}
	}

	return &stats
}

func startExec(context context.Context, config *model.Config, localPathLookup func(fileID int) string, id int, name string, inputs *persist.Bindings, listenerUpdates chan *Update) string {
	listener := &execListener{ruleApplicationID: id, c: listenerUpdates}
	rule := config.Rules[name]
	executorName := rule.ExecutorName
	executor := config.Executors[executorName]
	builder := executor.Builder(id)
	localizedInputs := inputs.Transform(func(artifact *persist.Artifact) *persist.Artifact {
		return localizeArtifact(builder, artifact)
	})
	runStatements := expandRunStatements(rule.RunStatements, localizedInputs, rule.Outputs, localPathLookup, builder)
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
		artifacts[i] = artifactPropsFromJson(output, func(filename string) (int, error) {
			var fullPath string
			if path.IsAbs(filename) {
				fullPath = filename
			} else {
				fullPath = path.Join(workDir, filename)
			}

			log.Printf("path: %s", filename)
			log.Printf("workDir: %s", workDir)
			log.Printf("fullPath: %s", fullPath)
			return getFileID(fullPath)
		})
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

func ReplayAndExport(stateDir string, filename string) (graph *graph.Graph, db *persist.DB, err error) {
	config := model.NewConfig()
	// config.ReplayOnly = true
	config.StateDir = stateDir

	db = persist.NewDB(stateDir)
	db.DisableUpdates()

	err = parseFile(config, filename)
	if err != nil {
		db.Close()
		return nil, nil, err
	}

	graph, _ = runAndGetGraph(context.Background(), config, db)
	return graph, db, nil
}

func RunRulesInFile(stateDir string, filename string) (*RunStats, error) {
	config := model.NewConfig()
	config.StateDir = stateDir

	db := persist.NewDB(stateDir)

	config.Executors[model.DefaultExecutorName] = &executor.LocalExec{JobDir: stateDir}

	err := parseFile(config, filename)
	if err != nil {
		return nil, err
	}

	_, stats := runAndGetGraph(context.Background(), config, db)
	return stats, nil
}
