package goconseq

import (
	"context"
	"log"
	"testing"
)

func mkstrmap(name string, value string) map[string]string {
	pps := make(map[string]string)
	pps[name] = value
	return pps
}

func makeConfig() *Config {
	config := NewConfig()
	config.AddRule(&Rule{Name: "r1",
		Query:   nil,
		Outputs: []map[string]string{mkstrmap("prop1", "value1")}})
	return config
}

func TestSimpleRun(t *testing.T) {
	config := makeConfig()
	log.Printf("test")
	run(context.Background(), config)
}
