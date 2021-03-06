package run

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/pgm/goconseq/executor"
	"github.com/pgm/goconseq/model"
	"github.com/pgm/goconseq/parser"
	"github.com/pgm/goconseq/persist"
	"github.com/stretchr/testify/assert"
)

func mkstrmap(name string, value string) map[string]string {
	pps := make(map[string]string)
	pps[name] = value
	return pps
}

type MockExecutor struct {
	db         *persist.DB
	resultBody string
}

type MockExecutionBuilder struct {
	executor *MockExecutor
	id       int
}

type MockExecution struct {
	executor *MockExecutor
	id       int
}

func (m *MockExecutor) Resume(resumeState string) (exec model.Execution, err error) {
	panic("unimp")
}

func (m *MockExecutor) Builder(id int) model.ExecutionBuilder {
	return &MockExecutionBuilder{executor: m, id: id}
}

func (m *MockExecutionBuilder) Localize(fileId int) (string, error) {
	panic("unimp")
}

func (m *MockExecutionBuilder) AddFile(body []byte) (string, error) {
	panic("unimp")
}

func (m *MockExecutionBuilder) Prepare(stmts []*model.RunWithStatement) error {
	workDir := m.executor.db.GetWorkDir(m.id)
	os.MkdirAll(workDir, os.ModePerm)
	if len(stmts) != 0 {
		panic("unimp")
	}
	return nil
}

func (m *MockExecutionBuilder) Start(context context.Context) (exec model.Execution, err error) {
	return &MockExecution{executor: m.executor, id: m.id}, nil
}

func (m *MockExecution) GetResumeState() string {
	return ""
}

func (m *MockExecution) Wait(listener model.Listener) {
	listener.UpdateStatus("executing")
	workDir := m.executor.db.GetWorkDir(m.id)
	os.MkdirAll(workDir, os.ModePerm)
	file, err := os.Create(path.Join(workDir, "results.json"))
	if err != nil {
		panic(err)
	}
	defer file.Close()
	_, err = file.Write([]byte(m.executor.resultBody))
	if err != nil {
		panic(err)
	}

	listener.Completed(&model.CompletionState{Success: true})
}

func TestExpandTemplates(t *testing.T) {
	bindings := persist.NewBindings()
	props := persist.NewArtifactProperties()
	props.Strings["c"] = "d"
	bindings.AddArtifact("b", &persist.Artifact{Properties: props})
	s := expandTemplate("inputs.b.c = {{ inputs.b.c }}", bindings)
	assert.Equal(t, "inputs.b.c = d", s)
}

func TestSimpleSingleRuleRun(t *testing.T) {
	stateDir, err := ioutil.TempDir("", "TestSimpleSingleRuleRun")
	if err != nil {
		panic(err)
	}

	config := model.NewConfig()
	config.StateDir = stateDir
	config.AddRule(&model.Rule{Name: "r1",
		Query: nil,
		// todo: rules without any defined outputs appear to not be included in graph?
		// todo add support for ExpectedOutput
		ExpectedOutputs: []*model.QueryTemplate{
			&model.QueryTemplate{
				[]*model.TemplateProperty{&model.TemplateProperty{Name: "prop", Value: "value"}}}},
		//		Outputs:      []map[string]string{mkstrmap("prop1", "value1")},
		ExecutorName: model.DefaultExecutorName})
	db := persist.NewDB(stateDir)
	config.Executors[model.DefaultExecutorName] = &MockExecutor{db: db, resultBody: `{"outputs": [{"prop": "value"}]}`}

	run(context.Background(), config, db)
	artifacts := db.FindArtifacts(map[string]string{})
	assert.Equal(t, 1, len(artifacts))
	db.Close()
}

func TestReplay(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	assert.Nil(t, err)
	defer os.RemoveAll(stateDir)

	aAndXrules := `
	rule a:
		outputs: {'type': 'a-out'}
		run 'date'

	rule x:
		inputs: a={'type': 'a-out'}
		outputs: {'type': 'x-out', 'value': '1'}, {'type': 'x-out', 'value': '2'}
		run 'date'

	`
	yRule := `
	rule y:
		inputs: x={'type': 'x-out'}
		outputs: {'type': 'y-out', 'parent':'{{ inputs.x.value }}'}
		run 'date'
	`
	allRules := aAndXrules + yRule

	db, config := parseRules(stateDir, allRules)
	config.Executors[model.DefaultExecutorName] = &executor.LocalExec{JobDir: stateDir}
	run(context.Background(), config, db)
	db.Close()

	checkWithPartialRules := func(rules string, expectedA int, expectedX int, expectedY int) {
		// now, reopen db in replay-only mode
		db, config = parseRules(stateDir, rules)
		db.DisableUpdates()
		config.ReplayOnly = true
		run(context.Background(), config, db)
		// verify we can see only a and x artifacts
		aOut := db.FindArtifacts(map[string]string{"type": "a-out"})
		xOut := db.FindArtifacts(map[string]string{"type": "x-out"})
		yOut := db.FindArtifacts(map[string]string{"type": "y-out"})
		assert.Equal(t, expectedA, len(aOut))
		assert.Equal(t, expectedX, len(xOut))
		assert.Equal(t, expectedY, len(yOut))
		db.Close()
	}

	checkWithPartialRules(aAndXrules, 1, 2, 0)
	// this time, only y and we shouldn't see anything because we're missing root
	checkWithPartialRules(yRule, 0, 0, 0)
	// Now, with all rules, we should see everything
	checkWithPartialRules(allRules, 1, 2, 2)

}
func run(ctx context.Context, config *model.Config, db *persist.DB) *RunStats {
	_, stats := runAndGetGraph(ctx, config, db)
	return stats
}

func parseRules(stateDir string, rules string) (*persist.DB, *model.Config) {
	config := model.NewConfig()
	config.StateDir = stateDir
	db := persist.NewDB(stateDir)
	config.Executors[model.DefaultExecutorName] = &MockExecutor{db: db, resultBody: `{"outputs": [{"prop": "value"}]}`}

	statements, err := parser.ParseString(rules)
	if err != nil {
		panic(err)
	}
	statements.Eval(config)
	return db, config
}

func TestRun3RuleChain(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	assert.Nil(t, err)
	defer os.RemoveAll(stateDir)

	db, config := parseRules(stateDir, `
		rule a:
			outputs: {'type': 'a-out'}
			run 'date'

		rule x:
			inputs: a={'type': 'a-out'}
			outputs: {'type': 'x-out', 'value': '1'}, {'type': 'x-out', 'value': '2'}
			run 'date'

		rule y:
			inputs: x={'type': 'x-out'}
			outputs: {'type': 'y-out', 'parent':'{{ inputs.x.value }}'}
			run 'date'
	`)
	config.Executors[model.DefaultExecutorName] = &executor.LocalExec{JobDir: stateDir}
	run(context.Background(), config, db)
	aOut := db.FindArtifacts(map[string]string{"type": "a-out"})
	xOut := db.FindArtifacts(map[string]string{"type": "x-out"})
	yOut := db.FindArtifacts(map[string]string{"type": "y-out"})
	assert.Equal(t, 1, len(aOut))
	assert.Equal(t, 2, len(xOut))
	assert.Equal(t, 2, len(yOut))
	parentValues := []string{yOut[0].Properties.Strings["parent"], yOut[1].Properties.Strings["parent"]}
	assert.ElementsMatch(t, []string{"1", "2"}, parentValues)
	db.Close()
}

func setupLocalExec(config *model.Config, stateDir string) *executor.LocalExec {
	e := &executor.LocalExec{JobDir: stateDir}
	config.Executors[model.DefaultExecutorName] = e
	return e
}

func TestConflictingOutputs(t *testing.T) {
	// make sure that if we have two rules make the same output, it's considered a failure
	stateDir, err := ioutil.TempDir("", t.Name())
	assert.Nil(t, err)
	defer os.RemoveAll(stateDir)

	initialTwoRules := `
	rule a1:
		outputs: {'type': 'a-out'}

	rule a2:
		outputs: {'type': 'a-out'}

	rule b:
		inputs: a={'type': 'a-out'}
		outputs: {'type': 'b-out'}
	`

	db, config := parseRules(stateDir, initialTwoRules)
	setupLocalExec(config, stateDir)

	stats := run(context.Background(), config, db)
	assert.Equal(t, 3, stats.Executions)
	assert.Equal(t, 2, stats.SuccessfulCompletions)
	assert.Equal(t, 1, stats.FailedCompletions)
	assert.Equal(t, 0, stats.ExistingAppliedRules)
}

func TestRuleWithAllQuery(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	assert.Nil(t, err)
	defer os.RemoveAll(stateDir)

	initialTwoRules := `
	rule a1:
		outputs: {'type': 'a-out', 'value': '1'}

	rule a2:
		outputs: {'type': 'a-out', 'value': '2'}

	rule b:
		inputs: a= all {'type': 'a-out'}
		outputs: {'type': 'b-out'}
	`

	db, config := parseRules(stateDir, initialTwoRules)
	setupLocalExec(config, stateDir)

	stats := run(context.Background(), config, db)
	assert.Equal(t, 3, stats.Executions)
	assert.Equal(t, 3, stats.SuccessfulCompletions)
	assert.Equal(t, 0, stats.FailedCompletions)
	assert.Equal(t, 0, stats.ExistingAppliedRules)

	aOut := db.FindArtifacts(map[string]string{"type": "a-out"})
	assert.Equal(t, 2, len(aOut))
	bOut := db.FindArtifacts(map[string]string{"type": "b-out"})
	assert.Equal(t, 1, len(bOut))
}

func TestRunTwice(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(stateDir)

	initialTwoRules := `
		rule a:
			outputs: {'type': 'a-out'}

		rule b:
			inputs: a={'type': 'a-out'}
			outputs: {'type': 'b-out'}
	`

	db, config := parseRules(stateDir, initialTwoRules)
	setupLocalExec(config, stateDir)

	stats := run(context.Background(), config, db)
	assert.Equal(t, 2, stats.Executions)
	assert.Equal(t, 2, stats.SuccessfulCompletions)
	assert.Equal(t, 0, stats.ExistingAppliedRules)

	aOut := db.FindArtifacts(map[string]string{"type": "a-out"})
	bOut := db.FindArtifacts(map[string]string{"type": "b-out"})
	assert.Equal(t, 1, len(aOut))
	assert.Equal(t, 1, len(bOut))
	db.Close()

	// reopen db and execute the same rules. Should be a no-op
	db, config = parseRules(stateDir, initialTwoRules)
	stats = run(context.Background(), config, db)
	assert.Equal(t, 0, stats.Executions)
	assert.Equal(t, 0, stats.SuccessfulCompletions)
	assert.Equal(t, 2, stats.ExistingAppliedRules)
	db.Close()
}

func TestRunChangedRules(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(stateDir)

	initialTwoRules := `
		rule a:
			outputs: {'type': 'a-out'}

		rule b1:
			inputs: a={'type': 'a-out'}
			outputs: {'type': 'b-out', 'value': '1'}
	`

	db, config := parseRules(stateDir, initialTwoRules)
	setupLocalExec(config, stateDir)

	stats := run(context.Background(), config, db)
	assert.Equal(t, 2, stats.Executions)
	assert.Equal(t, 2, stats.SuccessfulCompletions)
	assert.Equal(t, 0, stats.ExistingAppliedRules)
	db.Close()

	// reopen db and execute the same rules. Should be a no-op
	db, config = parseRules(stateDir, `
	rule a:
		outputs: {'type': 'a-out'}

	rule b2:
		inputs: a={'type': 'a-out'}
		outputs: {'type': 'b-out', 'value': '2'}
`)
	setupLocalExec(config, stateDir)
	stats = run(context.Background(), config, db)
	assert.Equal(t, 1, stats.Executions)
	assert.Equal(t, 1, stats.SuccessfulCompletions)
	assert.Equal(t, 1, stats.ExistingAppliedRules)

	bOut := db.FindArtifacts(map[string]string{"type": "b-out"})
	assert.Equal(t, 1, len(bOut))
	assert.Equal(t, "2", bOut[0].Properties.Strings["value"])
	db.Close()
}

func TestInitialArtifact(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(stateDir)

	initialRules := `
		add-if-missing {'type': 'a-out', 'value': '1'}

		rule b1:
			inputs: a={'type': 'a-out'}
			outputs: {'type': 'b-out'}
	`

	db, config := parseRules(stateDir, initialRules)
	setupLocalExec(config, stateDir)

	stats := run(context.Background(), config, db)
	assert.Equal(t, 2, stats.Executions)
	assert.Equal(t, 2, stats.SuccessfulCompletions)
	assert.Equal(t, 0, stats.ExistingAppliedRules)
	db.Close()

	// reopen db and execute a modified rule which should result in both running
	db, config = parseRules(stateDir, `
	add-if-missing {'type': 'a-out', 'value': '2'}

	rule b1:
		inputs: a={'type': 'a-out'}
		outputs: {'type': 'b-out'}
	`)

	setupLocalExec(config, stateDir)
	stats = run(context.Background(), config, db)
	assert.Equal(t, 2, stats.Executions)
	assert.Equal(t, 2, stats.SuccessfulCompletions)
	assert.Equal(t, 0, stats.ExistingAppliedRules)
}

func writeFile(filename string, content string) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		panic(err)
	}
	f.Close()

}

func TestOutputFileRef(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		panic(err)
	}
	log.Printf("stateDir: %s", stateDir)
	defer os.RemoveAll(stateDir)

	db, config := parseRules(stateDir, `
		rule x:
			outputs: {'type': 'file', 'filename': {'$filename': 'out'}}
			run "touch out"	
	`)
	setupLocalExec(config, stateDir)

	stats := run(context.Background(), config, db)

	assert.Equal(t, 1, stats.SuccessfulCompletions)

	file := db.FindArtifacts(map[string]string{"type": "file"})
	assert.Equal(t, 1, len(file))
	fileID := file[0].Properties.Files["filename"]
	assert.Greater(t, fileID, 0)
}

type LocalFileLocalizer struct {
	db *persist.DB
}

func (m *LocalFileLocalizer) EnsureLocallyAccessible(fileID int) (string, error) {
	file := m.db.GetFile(fileID)
	return file.LocalPath, nil
}

func (m *LocalFileLocalizer) EnsureGloballyAccessible(fileID int) (string, error) {
	panic("unimp")
}

func TestInputFileRef(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		panic(err)
	}
	log.Printf("stateDir: %s", stateDir)
	defer os.RemoveAll(stateDir)

	// create sample file
	inputFileName := path.Join(stateDir, "sample")
	writeFile(inputFileName, "{\"outputs\": [{\"type\": \"fromfile\"}]}")
	log.Printf("wrote to %s", inputFileName)

	rules := fmt.Sprintf(`
		rule f:
			inputs: src=filename("%s")
		run "cp {{inputs.src.filename}} results.json"
	`, inputFileName)

	db, config := parseRules(stateDir, rules)
	e := setupLocalExec(config, stateDir)
	e.Files = &LocalFileLocalizer{db}

	stats := run(context.Background(), config, db)
	assert.Equal(t, 2, stats.Executions)
	assert.Equal(t, 2, stats.SuccessfulCompletions)
	assert.Equal(t, 0, stats.ExistingAppliedRules)

	bOut := db.FindArtifacts(map[string]string{"type": "fromfile"})
	assert.Equal(t, 1, len(bOut))

	db.Close()

	// reopen db and execute the same rules. Should be a no-op
	db, config = parseRules(stateDir, rules)

	setupLocalExec(config, stateDir)
	stats = run(context.Background(), config, db)
	assert.Equal(t, 0, stats.Executions)
	assert.Equal(t, 0, stats.SuccessfulCompletions)
	assert.Equal(t, 2, stats.ExistingAppliedRules)

	db.Close()

	// mutate the file and verify the rule gets re-run
	writeFile(inputFileName, "{\"outputs\": [{\"type\": \"fromfile2\"}]}")
	db, config = parseRules(stateDir, rules)

	e = setupLocalExec(config, stateDir)
	e.Files = &LocalFileLocalizer{db}
	stats = run(context.Background(), config, db)
	assert.Equal(t, 2, stats.Executions)
	assert.Equal(t, 2, stats.SuccessfulCompletions)
	assert.Equal(t, 0, stats.ExistingAppliedRules)

	bOut = db.FindArtifacts(map[string]string{"type": "fromfile"})
	assert.Equal(t, 0, len(bOut))
	bOut = db.FindArtifacts(map[string]string{"type": "fromfile2"})
	assert.Equal(t, 1, len(bOut))

	db.Close()
}
