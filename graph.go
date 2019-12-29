package goconseq

type propPair struct {
	name  string
	value string
}

type propPairs struct {
	//	pairs []*propPair
	pairSet map[propPair]bool
}

func (pp *propPairs) Add(pair propPair) {
	if pp.pairSet == nil {
		pp.pairSet = make(map[propPair]bool)
	}
	_, ok := pp.pairSet[pair]
	if !ok {
		pp.pairSet[pair] = true
	}
}

func (pp *propPairs) Has(pair propPair) bool {
	return pp.pairSet[pair]
}

func (pp *propPairs) Contains(other *propPairs) bool {
	for pair, _ := range other.pairSet {
		if !pp.Has(pair) {
			return false
		}
	}
	return true
}

type rule struct {
	name     string
	produces []*artifact
	consumes []*artifactRel
}

type artifactRel struct {
	isAll    bool
	artifact *artifact
}

type artifact struct {
	props      *propPairs
	consumedBy []*rule
	producedBy []*rule
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
}

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

func (a *artifactIndex) Find(queryProps *propPairs) []*artifact {
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

	// then for each consume relationship, find all matching artifacts and update the rule's consumes list
	for name, r := range g.ruleByName {
		rels, ok := g.consumeRels[name]
		if !ok {
			roots = append(roots, r)
		} else {
			for _, rel := range rels {
				for _, match := range index.Find(rel.artifact.props) {
					r.consumes = append(r.consumes, &artifactRel{isAll: rel.isAll, artifact: match})
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

// AddRuleConsumes records the given rule consumes the artifacts with the given properties
func (g *GraphBuilder) AddRuleConsumes(name string, isAll bool, props *propPairs) {
	if _, ok := g.ruleByName[name]; !ok {
		g.ruleByName[name] = newRule(name)
	}

	rels, ok := g.consumeRels[name]
	if !ok {
		rels = make([]*artifactRel, 0, 1)
	}
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
func (g *GraphBuilder) AddRuleProduces(name string, props *propPairs) {
	r, ok := g.ruleByName[name]
	if !ok {
		r = newRule(name)
		g.ruleByName[name] = r
	}
	r.produces = append(r.produces, &artifact{
		props:      props,
		consumedBy: make([]*rule, 0, 1),
		producedBy: make([]*rule, 0, 1)})
}

func ConstructExecutionPlan(g *Graph) *ExecutionPlan {
	// TODO: Doesn't support "forall". Revisit considering using "group by" instead of all
	plan := NewExecutionPlan()
	for _, rule := range g.roots {
		for _, artifact := range rule.produces {
			for _, subsequentRule := range artifact.consumedBy {
				// precursor string, successor string, waitForAll bool
				plan.AddDependency(rule.name, subsequentRule.name, false)
			}
		}
	}
	return plan
}

// func testLinearGraph() {
// 	gb := NewGraphBuilder()
// 	gb.AddRuleProduces("r1", parseProps("p:a1"))
// 	gb.AddRuleConsumes("r2", false, parseProps("p:a1"))
// 	gb.AddRuleProduces("r2", parseProps("p:a2"))
// 	gb.AddRuleConsumes("r3", false, parseProps("p:a3"))
// 	g := gb.Build()
// 	assert(len(g.roots) == 1)
// }

// func testForkJoinGraph() {
// 	gb := NewGraphBuilder()
// 	gb.AddRuleProduces("r1", parseProps("p1:a1", "p2:a1"))
// 	gb.AddRuleConsumes("r2", false, parseProps("p1:a1"))
// 	gb.AddRuleProduces("r2", parseProps("p2:a2"))
// 	gb.AddRuleConsumes("r3", false, parseProps("p2:a1"))
// 	gb.AddRuleProduces("r3", parseProps("p2:a2"))
// 	gb.AddRuleConsumes("r4", false, parseProps("p2:a2"))
// 	g := gb.Build()
// 	assert(len(g.roots) == 1)
// }

//////
