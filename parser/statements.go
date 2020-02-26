package parser

import (
	"fmt"

	"github.com/pgm/goconseq/model"
	"github.com/pgm/goconseq/persist"
)

type Statement interface {
	Eval(config *model.Config) error
}

type UnresolvedOutputProperty struct {
	Name       string
	IsFilename bool
	Value      string
}

type RuleStatementOutput struct {
	Properties []UnresolvedOutputProperty
}

type RuleStatement struct {
	Name              string
	Inputs            map[string]*model.InputQuery
	Outputs           []RuleStatementOutput
	ExecutorName      string
	RequiredResources map[string]float64
	RunStatements     []*model.RunWithStatement
}

func makeRuleOutput(output RuleStatementOutput) model.RuleOutput {
	properties := make([]model.RuleOutputProperty, len(output.Properties))
	for i, property := range output.Properties {
		properties[i] = model.RuleOutputProperty{Name: property.Name, Value: property.Value, IsFilename: property.IsFilename}
	}
	return model.RuleOutput{Properties: properties}
}

func (s *RuleStatement) Eval(config *model.Config) error {
	query := persist.QueryFromMaps(s.Inputs)
	outputs := make([]model.RuleOutput, len(s.Outputs))
	if s.Outputs == nil {
		outputs = nil
	} else {
		for i, output := range s.Outputs {
			outputs[i] = makeRuleOutput(output)
		}
	}
	config.AddRule(&model.Rule{Name: s.Name,
		Query:             query,
		Outputs:           outputs,
		ExecutorName:      s.ExecutorName,
		RequiredResources: s.RequiredResources,
		RunStatements:     s.RunStatements})
	return nil
}

type LetStatement struct {
	Name  string
	Value string
}

func (s *LetStatement) Eval(config *model.Config) error {
	if existingValue, exists := config.Vars[s.Name]; exists {
		return fmt.Errorf("Cannot define %s as %s (already defined as %s)", s.Name, s.Value, existingValue)
	}
	config.Vars[s.Name] = s.Value
	return nil
}

type ArtifactStatement struct {
	Artifact map[string]model.ArtifactValue
}

func (s *ArtifactStatement) Eval(config *model.Config) error {
	config.Artifacts = append(config.Artifacts, s.Artifact)
	return nil
}

type Statements struct {
	Statements []Statement
}

func (s *Statements) Eval(config *model.Config) error {
	for _, stmt := range s.Statements {
		err := stmt.Eval(config)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Statements) Add(stmt Statement) {
	s.Statements = append(s.Statements, stmt)
}
