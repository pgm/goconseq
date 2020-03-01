package graph

import (
	"fmt"
	"io"
)

func (g *Graph) PrintGraph(writer io.Writer) {
	ruleToID := make(map[*Rule]int)
	artifactToID := make(map[*artifact]int)
	nextID := 1

	// enuerate rules and assign an id to each
	g.ForEachRule(func(r *Rule) {
		ruleToID[r] = nextID
		nextID += 1
	})

	g.ForEachArtifact(func(a *artifact) {
		artifactToID[a] = nextID
		nextID += 1
	})

	fmt.Fprintf(writer, "digraph {\n")
	for r := range ruleToID {
		fmt.Fprintf(writer, "rule_%x [label=\"%s\"];\n", ruleToID[r], r.name)
	}
	for a := range artifactToID {
		typeValue := a.props.Get("type")
		if typeValue == "" {
			typeValue = "artifact"
		}
		fmt.Fprintf(writer, "artifact_%x [label=\"%s\", shape=box];\n", artifactToID[a], typeValue)
	}

	// now edges
	for r := range ruleToID {
		for _, a := range r.produces {
			fmt.Fprintf(writer, "rule_%x -> artifact_%x ;\n", ruleToID[r], artifactToID[a])
		}
		for _, ar := range r.consumes {
			fmt.Fprintf(writer, "artifact_%x -> rule_%x ;\n", artifactToID[ar.artifact], ruleToID[r])
		}
	}

	fmt.Fprintf(writer, "}\n")
}
