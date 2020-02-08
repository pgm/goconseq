package model

const FileRefType = "$filename_ref"

type Config struct {
	Rules map[string]*Rule
	Vars  map[string]string
	//	Artifacts []model.PropPairs
	Executors map[string]Executor
	StateDir  string
	Artifacts []map[string]string
}

func NewConfig() *Config {
	c := &Config{Rules: make(map[string]*Rule),
		Vars:      make(map[string]string),
		Executors: make(map[string]Executor)}

	return c
}

func (c *Config) AddRule(rule *Rule) {
	if rule.ExpectedOutputs != nil && rule.Outputs != nil {
		panic("Cannot have both expected outputs and constant outputs defined for rule")
	}
	c.Rules[rule.Name] = rule
}

type FileRepository interface {
	AddFileOrFind(localPath string, sha256 string) int
}
