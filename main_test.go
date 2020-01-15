package goconseq

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"

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
	panic("unimp")
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
	stateDir, err := ioutil.TempDir("", "TestSimpleSingleRuleRun")
	if err != nil {
		panic(err)
	}

	db, config := parseRules(stateDir, `
		rule a:
			outputs: {'type': 'a-out'}
			run 'date'

		rule x:
			inputs: a={'type': 'a-out'}
			outputs: {'type': 'x-out', 'value': '1'}, {'type': 'x-out', 'value': '2'}
			run 'date'

		rule y:
			inputs: in={'type': 'x-out'}
			outputs: {'type': 'y-out'}
			run 'date'
	`)
	config.Executors[model.DefaultExecutorName] = &LocalExec{jobDir: stateDir}
	run(context.Background(), config, db)
	aOut := db.FindArtifacts(map[string]string{"type": "a-out"})
	xOut := db.FindArtifacts(map[string]string{"type": "x-out"})
	yOut := db.FindArtifacts(map[string]string{"type": "y-out"})
	assert.Equal(t, 1, len(aOut))
	assert.Equal(t, 2, len(xOut))
	assert.Equal(t, 2, len(yOut))
}
