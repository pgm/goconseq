package graph

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func parseProps(txts ...string) *PropertiesTemplate {
	pps := &PropertiesTemplate{}
	for _, txt := range txts {
		pairStr := strings.Split(txt, ":")
		pps.AddConstantProperty(pairStr[0], pairStr[1])
	}
	return pps
}

func TestPropContains(t *testing.T) {
	pps1 := parseProps("p:a")
	pps2 := parseProps("p:a")
	pps3 := parseProps("p:a", "p:b")

	if !pps1.Contains(pps2) {
		t.Error("! pps1.Contains(pps2)")
	}
	if !pps2.Contains(pps1) {
		t.Error("! pps2.Contains(pps1)")
	}
	if !pps3.Contains(pps1) {
		t.Error("! pps3.Contains(pps1)")
	}
	if pps1.Contains(pps3) {
		t.Error("pps1.Contains(pps3)")
	}
}

func TestMinGraph(t *testing.T) {
	gb := NewGraphBuilder()
	gb.AddRule("r1")
	gb.AddRuleProduces("r1", parseProps("p:a"))
	gb.AddRule("r2")
	gb.AddRuleConsumes("r2", false, parseProps("p:a"))
	g := gb.Build()

	assert.Equal(t, 1, len(g.roots))

	r1 := g.roots[0]
	assert.Equal(t, len(r1.consumes), 0)
	assert.Equal(t, len(r1.produces), 1)
	a1 := r1.produces[0]
	assert.Equal(t, len(a1.producedBy), 1)
	assert.Equal(t, a1.producedBy[0], r1)
	assert.Equal(t, len(a1.consumedBy), 1)
	r2 := a1.consumedBy[0]
	assert.Equal(t, len(r2.consumes), 1)
	assert.Equal(t, len(r2.produces), 0)
}

func TestGraphWithNoOutput(t *testing.T) {
	gb := NewGraphBuilder()
	gb.AddRule("r1")
	g := gb.Build()

	assert.Equal(t, len(g.roots), 1)

	r1 := g.roots[0]
	assert.Equal(t, len(r1.consumes), 0)
	assert.Equal(t, len(r1.produces), 0)
}
