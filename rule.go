package goconseq

import (
	"./model"
	"./persist"
)

type RunWithStatement struct {
	Script     string
	Executable string
}

type Rule struct {
	Name              string
	Query             *persist.Query
	Outputs           []*model.PropPairs // change this to be optional and templates
	ExecutorName      string
	RequiredResources map[string]float64
	RunStatements     []*RunWithStatement
}

func (r *Rule) GetQueryProps() []*model.PropPairs {
	if r.Query == nil {
		return nil
	}
	return r.Query.GetProps()
}

func (r *Rule) GetOutputProps() []*model.PropPairs {
	return r.Outputs
}
