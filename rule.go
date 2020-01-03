package goconseq

import (
	"./graph"
	"./persist"
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

func (r *Rule) GetQueryProps() []*graph.ArtifactTemplate {
	if r.Query == nil {
		return nil
	}
	return r.Query.GetProps()
}

func (r *Rule) GetOutputProps() []*graph.ArtifactTemplate {
	templates := make([]*graph.ArtifactTemplate, 0, len(r.Outputs))
	for i, output := range r.Outputs {
		template := graph.ArtifactTemplate{}
		for k, v := range output {
			template.AddConstProperty(k, v)
		}
		templates = append(templates, template)
	}
	return templates
}
