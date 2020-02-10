package parser

import (
	"log"
	"testing"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	"github.com/pgm/goconseq/model"
	"github.com/stretchr/testify/assert"
)

func TestParseLetStatement(t *testing.T) {
	log.Printf("%v", &antlr.Set{})
	stmts, err := ParseString("let x = 'a'")
	assert.Nil(t, err)
	assert.Equal(t, len(stmts.Statements), 1)
	config := model.NewConfig()
	stmts.Eval(config)
	assert.Equal(t, config.Vars["x"], "a")
}

func TestParseAddIfMissing(t *testing.T) {
	log.Printf("%v", &antlr.Set{})
	stmts, err := ParseString("add-if-missing {'x': 'b'}")
	assert.Nil(t, err)
	assert.Equal(t, len(stmts.Statements), 1)
	config := model.NewConfig()
	stmts.Eval(config)
	assert.Equal(t, 1, len(config.Artifacts))
	artifact := config.Artifacts[0]
	assert.Equal(t, artifact["x"], "b")
}
func TestParseRule(t *testing.T) {
	log.Printf("%v", &antlr.Set{})
	stmts, err := ParseString("rule x: inputs: a={'type': 'banana'} outputs: {'type': 'out'}")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(stmts.Statements))
	stmt := stmts.Statements[0].(*RuleStatement)
	assert.Equal(t, "x", stmt.Name)
	outputType := ""
	for _, prop := range stmt.Outputs[0].Properties {
		if prop.Name == "type" {
			outputType = prop.Value
		}
	}
	assert.Equal(t, "out", outputType)
	assert.Equal(t, "banana", stmt.Inputs["a"]["type"])
}

func TestParseFailure(t *testing.T) {
	log.Printf("%v", &antlr.Set{})
	stmts, err := ParseString("let x = x")
	assert.Nil(t, stmts)
	assert.NotNil(t, err)
}

func TestParseRuleWithFileRef(t *testing.T) {
	log.Printf("%v", &antlr.Set{})
	stmts, err := ParseString("rule x: inputs: a=filename('sample') outputs: {'type': 'out'}")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(stmts.Statements))
	stmt := stmts.Statements[0].(*RuleStatement)
	assert.Equal(t, "x", stmt.Name)
	assert.Equal(t, "$filename_ref", stmt.Inputs["a"]["type"])
	assert.Equal(t, "sample", stmt.Inputs["a"]["filename"])
}
