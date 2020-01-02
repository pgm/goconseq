package goconseq

import (
	"context"
	"log"
	"testing"

	"./model"
)

func pp(name string, value string) *model.PropPairs {
	pps := &model.PropPairs{}
	pps.Add(model.PropPair{Name: name, Value: value})
	return pps
}

func makeConfig() *Config {
	config := NewConfig()
	config.AddRule(&Rule{Name: "r1",
		Query:   nil,
		Outputs: []*model.PropPairs{pp("prop1", "value1")}})
	return config
}

func TestSimpleRun(t *testing.T) {
	config := makeConfig()
	log.Printf("test")
	run(context.Background(), config)
}
