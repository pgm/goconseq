package model

import (
	"encoding/json"
	"log"

	"github.com/pgm/goconseq/graph"
	"github.com/pgm/goconseq/persist"
)

type RunWithStatement struct {
	Script     string
	Executable string
}

type TemplateProperty struct {
	Name    string
	Value   string
	NoValue bool
}

type QueryTemplate struct {
	Properties []*TemplateProperty
}

type Rule struct {
	Name              string
	Query             *persist.Query
	ExpectedOutputs   []*QueryTemplate
	Outputs           []RuleOutput
	ExecutorName      string
	RequiredResources map[string]float64
	RunStatements     []*RunWithStatement
}

type HasAsDict interface {
	AsDict() map[string]interface{}
}

func asDictSlice(v []interface{}) []interface{} {
	nv := make([]interface{}, len(v))
	for i := range v {
		nv[i] = v[i].(HasAsDict).AsDict()
	}
	return nv
}

func (r *Rule) Hash() string {
	outputs := make([]interface{}, len(r.Outputs))
	for i := range r.Outputs {
		ro := r.Outputs[i]
		outputs[i] = ro.AsDicts()
	}
	log.Printf("Warning: Rule.Hash() is incomplete")
	flat := map[string]interface{}{"name": r.Name,
		"query":   r.Query.AsDict(),
		"outputs": outputs} //,
	//		"required_resources": r.RequiredResources,
	//		"run_statements":     asDictSlice(r.RunStatements)}

	b, err := json.Marshal(flat)
	if err != nil {
		panic(nil)
	}
	return string(b)
}

type RuleOutputProperty struct {
	Name   string
	FileID int
	Value  string
}

func (p *RuleOutputProperty) HasFileID() bool {
	return p.FileID != 0
}

func (p *RuleOutputProperty) HasValue() bool {
	return !p.HasFileID()
}

type RuleOutput struct {
	Properties []RuleOutputProperty
}

func (ro *RuleOutput) AsDicts() []map[string]interface{} {
	nv := make([]map[string]interface{}, len(ro.Properties))
	for i := range nv {
		nv[i] = map[string]interface{}{"Name": ro.Properties[i].Name,
			"FileID": ro.Properties[i].FileID,
			"Value":  ro.Properties[i].Value}
	}
	return nv
}

func (ro *RuleOutput) AddPropertyString(Name string, Value string) {
	ro.Properties = append(ro.Properties, RuleOutputProperty{Name: Name, Value: Value})
}
func (ro *RuleOutput) AddPropertyFileID(Name string, FileID int) {
	if FileID == 0 {
		panic("invalid fileid")
	}
	ro.Properties = append(ro.Properties, RuleOutputProperty{Name: Name, FileID: FileID})
}

func (r *Rule) GetQueryProps() []*graph.PropertiesTemplate {
	if r.Query == nil {
		return nil
	}
	return r.Query.GetProps()
}

func (r *Rule) GetOutputProps() []*graph.PropertiesTemplate {
	templates := make([]*graph.PropertiesTemplate, 0, len(r.Outputs)+len(r.ExpectedOutputs))

	if r.Outputs != nil {
		for _, output := range r.Outputs {
			template := graph.PropertiesTemplate{}
			for _, prop := range output.Properties {
				if prop.HasValue() {
					template.AddConstantProperty(prop.Name, prop.Value)
				}
			}
			templates = append(templates, &template)
		}
	} else {
		for _, inTemplate := range r.ExpectedOutputs {
			template := graph.PropertiesTemplate{}
			for _, prop := range inTemplate.Properties {
				if !prop.NoValue {
					template.AddConstantProperty(prop.Name, prop.Value)
				}
			}
			templates = append(templates, &template)
		}
	}

	return templates
}
