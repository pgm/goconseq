package goconseq

import (
	"context"
	"testing"

	"github.com/pgm/goconseq/model"
	"github.com/stretchr/testify/assert"
)

func mkstrmap(name string, value string) map[string]string {
	pps := make(map[string]string)
	pps[name] = value
	return pps
}

func TestSimpleSingleRuleRun(t *testing.T) {
	panic("broken")
	// config := model.NewConfig(&LocalExec{})
	// config.AddRule(&model.Rule{Name: "r1",
	// 	Query:   nil,
	// 	Outputs: []map[string]string{mkstrmap("prop1", "value1")}})
	// db := run(context.Background(), config)
	// artifacts := db.FindArtifacts(map[string]string{})
	// assert.Equal(t, 1, len(artifacts))
}

func parseRules(rules string) *model.Config {
	panic("broken")
	// config := model.NewConfig(&LocalExec{})
	// statements, err := parser.ParseString(rules)
	// if err != nil {
	// 	panic(err)
	// }
	// statements.Eval(config)
	// return config
}

func TestRun3RuleChain(t *testing.T) {
	config := parseRules(`
		rule a:
			outputs: {'type': 'a-out'}
			run: 'date'

		rule x:
			inputs: a={'type': 'a-out'}
			outputs: {'type': 'x-out', 'value': '1'}, {'type': 'x-out', 'value': '2'}
			run: 'date'

		rule y:
			inputs: in={'type': 'x-out'}
			outputs: {'type': 'y-out'}
			run: 'date'
	`)
	db := run(context.Background(), config)
	aOut := db.FindArtifacts(map[string]string{"type": "a-out"})
	xOut := db.FindArtifacts(map[string]string{"type": "x-out"})
	yOut := db.FindArtifacts(map[string]string{"type": "y-out"})
	assert.Equal(t, 1, len(aOut))
	assert.Equal(t, 2, len(xOut))
	assert.Equal(t, 2, len(yOut))
}
