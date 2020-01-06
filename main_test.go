package goconseq

import (
	"context"
	"testing"
)

func mkstrmap(name string, value string) map[string]string {
	pps := make(map[string]string)
	pps[name] = value
	return pps
}

func TestSimpleSingleRuleRun(t *testing.T) {
	config := NewConfig()
	config.AddRule(&Rule{Name: "r1",
		Query:   nil,
		Outputs: []map[string]string{mkstrmap("prop1", "value1")}})
	run(context.Background(), config)
}

// func TestRun3RuleChain(t *testing.T) {
// 	config := NewConfig()
// 	config.AddRule(&Rule{Name: "r1",
// 		Query:   nil,
// 		Outputs: []map[string]string{map[string]string{"prop": "1", "type": "a1"}, map[string]string{"prop": "2", "type": "a2"}}})
// 	config.AddRule(&Rule{Name: "r2",
// 		Query: &persist.Query{forEach: []*persist.QueryBinding{
// 			&persist.QueryBinding{bindingVariable: "in", constantConstraints: map[string]string{"type": "a1"}}}},
// 		Outputs: []map[string]string{mkstrmap("type", "a2")}})

// 	log.Printf("test")
// 	run(context.Background(), config)
// }
