package graph

import (
	"fmt"
	"log"
	"strings"
)

type rule struct {
	name     string
	produces []*artifact
	consumes []*artifactRel
}

type artifactRel struct {
	isAll    bool
	artifact *artifact
}

func (ar *artifactRel) String() string {
	return fmt.Sprintf("{isAll=%v, artifact=%s}", ar.isAll, ar.artifact.String())
}

// GraphBuilder is a data structure which is incrementally constructed via Add.. methods and then Build() can be called
type GraphBuilder struct {
	ruleByName  map[string]*rule
	consumeRels map[string][]*artifactRel
}

func NewGraphBuilder() *GraphBuilder {
	return &GraphBuilder{ruleByName: make(map[string]*rule),
		consumeRels: make(map[string][]*artifactRel)}
}

type Graph struct {
	roots []*rule
	// rulesByName map[string]*rule
}

// func (g *Graph) findUpstreamRules(name string) []string {
// 	result := make([]string, 0, 10)
// 	rule := g.rulesByName[name]
// 	for _, consume := range rule.consumes {
// 		if consume.isAll {
// 			panic("unimp")
// 		} else {
// 			upstreamRules := consume.artifact.producedBy
// 			for _, upstreamRule := range upstreamRules {
// 				result = append(result, upstreamRule.name)
// 				result = append(result, g.findUpstreamRules(upstreamRule.name)...)
// 			}
// 		}
// 	}
// 	return result
// }

// func (g *Graph) findDownstreamRules(name string) []string {
// 	result := make([]string, 0, 10)
// 	rule := g.rulesByName[name]
// 	for _, artifact := range rule.produces {
// 		for _, downstreamRule := range artifact.consumedBy {
// 			result = append(result, downstreamRule.name)
// 			result = append(result, g.findDownstreamRules(downstreamRule.name)...)
// 		}
// 	}
// 	return result
// }

// func (g *Graph) subgraphWithRules(ruleNames []string) *Graph {
// 	ruleNameSet := make(map[string]bool)
// 	for _, name := range ruleNames {
// 		ruleNameSet[name] = true
// 	}

// 	newRoots := make([]*rule, 0, len(g.roots))
// 	for _, rule := range g.roots {
// 		if
// 	}
// }

// func (g *Graph) Subset(ruleName string) *Graph {
// 	// finds the graph that should be executed if we want to restrict ourselves to just paths including ruleName
// 	ruleNames := g.findUpstreamRules(ruleName)
// 	ruleNames = append(ruleNames, g.findDownstreamRules(ruleName)...)
// 	return g.subgraphWithRules(ruleNames)
// }

func (g *Graph) ForEachRule(f func(r *rule)) {
	seen := make(map[*rule]bool)
	var traverse func(r *rule)
	traverse = func(r *rule) {
		seen[r] = true
		f(r)
		for _, a := range r.produces {
			for _, childRule := range a.consumedBy {
				if !seen[childRule] {
					traverse(childRule)
				}
			}
		}
	}
	for _, rule := range g.roots {
		traverse(rule)
	}
}

// func (g *Graph) Print() {
// 	printRule := func(r *rule) {
// 		for _, a := range r.consumes {
// 			log.Printf("consumes: %p", a)
// 		}
// 		for _, a := range r.consumes {
// 			log.Printf("produces: %p", a)
// 		}
// 	}
// 	g.ForEachRule(printRule)
// }

type artifactIndex struct {
	// naive implementation. replace with something more efficient
	artifacts []*artifact
}

func newArtifactIndex() *artifactIndex {
	return &artifactIndex{}
}

func (a *artifactIndex) Add(artifact *artifact) {
	a.artifacts = append(a.artifacts, artifact)
}

func (a *artifactIndex) String() string {
	sb := strings.Builder{}
	sb.WriteString("artifactIndex{")
	for _, artifact := range a.artifacts {
		sb.WriteString(artifact.String())
		sb.WriteString(",\n")
	}
	sb.WriteString("}")
	return sb.String()
}

func (a *artifactIndex) Find(queryProps *PropertiesTemplate) []*artifact {
	matches := make([]*artifact, 0)
	for _, candidate := range a.artifacts {
		if candidate.props.Contains(queryProps) {
			matches = append(matches, candidate)
		}
	}
	return matches
}

func (g *GraphBuilder) Build() *Graph {
	roots := make([]*rule, 0, 10)

	// construct index of all produced artifacts
	index := newArtifactIndex()
	for _, r := range g.ruleByName {
		for _, artifact := range r.produces {
			index.Add(artifact)
		}
	}
	log.Printf("Dumping index: %s", index.String())

	// then for each consume relationship, find all matching artifacts and update the rule's consumes list
	for name, r := range g.ruleByName {
		rels := g.consumeRels[name]
		if len(rels) == 0 {
			roots = append(roots, r)
		} else {
			for _, rel := range rels {
				log.Printf("consume rel %s: %s", name, rel.String())
				matches := index.Find(rel.artifact.props)
				if len(matches) > 0 {
					for _, match := range matches {
						r.consumes = append(r.consumes, &artifactRel{isAll: rel.isAll, artifact: match})
					}
				} else {
					log.Printf("Warning: %s will never execute because no artifact will be created that satisfies %s", name, rel.String())
				}
			}
		}
	}

	// now that the rules objects are fully populated, iterate through all the referenced artifacts and update the back refs
	// also, collect all of the roots so we can return that as the graph
	for _, r := range g.ruleByName {
		for _, artifact := range r.produces {
			artifact.producedBy = append(artifact.producedBy, r)
		}

		for _, rel := range r.consumes {
			rel.artifact.consumedBy = append(rel.artifact.consumedBy, r)
		}
	}

	return &Graph{roots}
}

func (g *GraphBuilder) AddRule(name string) {
	if _, ok := g.ruleByName[name]; !ok {
		g.ruleByName[name] = newRule(name)
		g.consumeRels[name] = make([]*artifactRel, 0, 1)
	}
}

// AddRuleConsumes records the given rule consumes the artifacts with the given properties
func (g *GraphBuilder) AddRuleConsumes(name string, isAll bool, props *PropertiesTemplate) {
	rels := g.consumeRels[name]
	g.consumeRels[name] = append(rels, &artifactRel{
		isAll:    isAll,
		artifact: &artifact{props: props}})
}
func newRule(name string) *rule {
	return &rule{name: name,
		produces: make([]*artifact, 0, 1),
		consumes: make([]*artifactRel, 0, 1)}
}

// AddRuleProduces records the given rule produces artifacts with the given properties
func (g *GraphBuilder) AddRuleProduces(name string, props *PropertiesTemplate) {
	r := g.ruleByName[name]
	r.produces = append(r.produces, &artifact{
		props:      props,
		consumedBy: make([]*rule, 0, 1),
		producedBy: make([]*rule, 0, 1)})
}
