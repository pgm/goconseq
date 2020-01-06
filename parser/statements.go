package parser

import "fmt"

type AddIfMissingStatement struct {
	Artifact map[string]string
}

func (s *AddIfMissingStatement) Eval(config *Config) error {
	panic("unimp")
}

type RuleStatement struct {
	Name    string
	Inputs  map[string]map[string]string
	Outputs []map[string]string
}

func (s *RuleStatement) Eval(config *Config) error {
	panic("unimp")
}

type LetStatement struct {
	Name  string
	Value string
}

func (s *LetStatement) Eval(config *Config) error {
	if existingValue, exists := config.Vars[s.Name]; exists {
		return fmt.Errorf("Cannot define %s as %s (already defined as %s)", s.Name, s.Value, existingValue)
	}
	config.Vars[s.Name] = s.Value
	return nil
}

type Statement interface {
	Eval(config *Config) error
}

func (s *Statements) Eval(config *Config) error {
	for _, stmt := range s.Statements {
		err := stmt.Eval(config)
		if err != nil {
			return err
		}
	}
	return nil
}

type Statements struct {
	Statements []Statement
}

func (s *Statements) Add(stmt Statement) {
	s.Statements = append(s.Statements, stmt)
}
