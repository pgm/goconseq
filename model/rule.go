package model

import (
	"encoding/json"
	"log"
	"sort"

	"github.com/pgm/goconseq/graph"
)

type InputQuery struct {
	IsAll      bool
	Properties map[string]string
}

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

type QueryI interface {
	AsDict() map[string]interface{}
	GetProps() []*graph.PropertiesTemplate
	IsEmpty() bool
	ExecuteQuery(db interface{}) []interface{}
}

type Rule struct {
	Name              string
	Query             QueryI
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

type indirectkey struct {
	originalIndex int
	key           string
}

func dictToStr(a interface{}) string {
	b, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func sortJsonList(l []interface{}) []interface{} {
	indirectkeys := make([]indirectkey, len(l))
	for i := range l {
		indirectkeys[i] = indirectkey{i, dictToStr(l[i])}
	}
	sort.Slice(indirectkeys, func(i, j int) bool {
		return (indirectkeys[i].key < indirectkeys[j].key)
	})

	sortedOutputs := make([]interface{}, len(l))
	for i := range l {
		sortedOutputs[i] = l[indirectkeys[i].originalIndex]
	}
	return sortedOutputs
}

func (r *Rule) Hash() string {
	outputs := make([]interface{}, len(r.Outputs))
	for i := range r.Outputs {
		ro := r.Outputs[i]
		outputs[i] = sortJsonList(ro.AsDicts())
	}

	log.Printf("Warning: Rule.Hash() is incomplete")
	flat := map[string]interface{}{"name": r.Name,
		"outputs": sortJsonList(outputs)} //,
	//		"required_resources": r.RequiredResources,
	//		"run_statements":     asDictSlice(r.RunStatements)}
	if r.Query != nil {
		flat["query"] = r.Query.AsDict()
	}

	b, err := json.Marshal(flat)
	if err != nil {
		panic(nil)
	}
	return string(b)
}

type RuleOutputProperty struct {
	Name       string
	IsFilename bool
	Value      string
}

// func (p *RuleOutputProperty) HasFileID() bool {
// 	return p.FileID != 0
// }

// func (p *RuleOutputProperty) HasValue() bool {
// 	return !p.HasFileID()
// }

type RuleOutput struct {
	Properties []RuleOutputProperty
}

func (ro *RuleOutput) AsDicts() []interface{} {
	nv := make([]interface{}, len(ro.Properties))
	for i := range nv {
		nv[i] = map[string]interface{}{"Name": ro.Properties[i].Name,
			"IsFilename": ro.Properties[i].IsFilename,
			"Value":      ro.Properties[i].Value}
	}
	return nv
}

func (ro *RuleOutput) AddPropertyString(Name string, Value string) {
	ro.Properties = append(ro.Properties, RuleOutputProperty{Name: Name, Value: Value})
}

func (ro *RuleOutput) AddPropertyFilename(Name string, Value string) {
	ro.Properties = append(ro.Properties, RuleOutputProperty{Name: Name, Value: Value, IsFilename: true})
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
				template.AddConstantProperty(prop.Name, prop.Value)
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
