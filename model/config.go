package model

type Config struct {
	Rules map[string]*Rule
	Vars  map[string]string
	//	Artifacts []model.PropPairs
	Executors map[string]Executor
	StateDir  string
}

func NewConfig() *Config {
	c := &Config{Rules: make(map[string]*Rule),
		Vars:      make(map[string]string),
		Executors: make(map[string]Executor)}

	return c
}

func (c *Config) AddRule(rule *Rule) {
	c.Rules[rule.Name] = rule
}
