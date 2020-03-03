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

func TestParseResultOutputs(t *testing.T) {
	outputs, err := parseResultsCharStream(antlr.NewInputStream(`[{"a": "b"}, {"c": {"$filename": "d"}}]`))
	assert.Nil(t, err)
	assert.Equal(t, len(outputs), 2)
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
	assert.Equal(t, artifact["x"], model.ArtifactValue{"b", false})
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
	assert.Equal(t, "banana", stmt.Inputs["a"].Properties["type"])
}

func TestParseFailure(t *testing.T) {
	log.Printf("%v", &antlr.Set{})
	stmts, err := ParseString("let x = x")
	assert.Nil(t, stmts)
	assert.NotNil(t, err)
}

func TestParseRuleWithFilenameInInputs(t *testing.T) {
	log.Printf("%v", &antlr.Set{})
	stmts, err := ParseString("rule x: inputs: a=filename('sample') outputs: {'type': 'out'}")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(stmts.Statements))
	stmt := stmts.Statements[0].(*RuleStatement)
	assert.Equal(t, "x", stmt.Name)
	assert.Equal(t, "$filename_ref", stmt.Inputs["a"].Properties["type"])
	assert.Equal(t, "sample", stmt.Inputs["a"].Properties["name"])
	astmt := stmts.Statements[1].(*ArtifactStatement)
	assert.True(t, astmt.Artifact["filename"].IsFilename)
	assert.False(t, astmt.Artifact["type"].IsFilename)
}

func TestParseRuleWithFileRefInOutputs(t *testing.T) {
	log.Printf("%v", &antlr.Set{})
	stmts, err := ParseString("rule x: outputs: { 'filename': {'$filename': 'x'}}")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(stmts.Statements))
	stmt := stmts.Statements[0].(*RuleStatement)
	assert.Equal(t, "x", stmt.Name)
	fileProp := stmt.Outputs[0].Properties[0]
	assert.Equal(t, "x", fileProp.Value)
	assert.Equal(t, "filename", fileProp.Name)
	assert.True(t, fileProp.IsFilename)
}
