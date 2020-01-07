package model

import (
	"github.com/pgm/goconseq/graph"
	"github.com/pgm/goconseq/persist"
)

type RunWithStatement struct {
	Script     string
	Executable string
}

type Rule struct {
	Name              string
	Query             *persist.Query
	Outputs           []map[string]string
	ExecutorName      string
	RequiredResources map[string]float64
	RunStatements     []*RunWithStatement
}

func (r *Rule) GetQueryProps() []*graph.PropertiesTemplate {
	if r.Query == nil {
		return nil
	}
	return r.Query.GetProps()
}

func (r *Rule) GetOutputProps() []*graph.PropertiesTemplate {
	templates := make([]*graph.PropertiesTemplate, 0, len(r.Outputs))
	for _, output := range r.Outputs {
		template := graph.PropertiesTemplate{}
		for k, v := range output {
			template.AddConstantProperty(k, v)
		}
		templates = append(templates, &template)
	}
	return templates
}
