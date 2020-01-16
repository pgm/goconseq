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
	stmt := stmts.Statements[0].(*AddIfMissingStatement)
	assert.Equal(t, stmt.Artifact["x"], "b")
}
func TestParseRule(t *testing.T) {
	log.Printf("%v", &antlr.Set{})
	stmts, err := ParseString("rule x: inputs: a={'type': 'banana'} outputs: {'type': 'out'}")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(stmts.Statements))
	stmt := stmts.Statements[0].(*RuleStatement)
	assert.Equal(t, "x", stmt.Name)
	assert.Equal(t, "out", stmt.Outputs[0]["type"])
	assert.Equal(t, "banana", stmt.Inputs["a"]["type"])
}

func TestParseFailure(t *testing.T) {
	log.Printf("%v", &antlr.Set{})
	stmts, err := ParseString("let x = x")
	assert.Nil(t, stmts)
	assert.NotNil(t, err)
}
