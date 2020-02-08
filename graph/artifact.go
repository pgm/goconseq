package graph

import "strings"

type stringProperty struct {
	Name  string
	Value string
}

type PropertiesTemplate struct {
	//	pairs []*propPair
	constProps      map[stringProperty]bool
	additionalProps map[string]bool
}

type artifact struct {
	props      *PropertiesTemplate
	consumedBy []*rule
	producedBy []*rule
}

func (pt *PropertiesTemplate) String() string {
	sb := strings.Builder{}
	for sp := range pt.constProps {
		sb.WriteString("{")
		sb.WriteString(sp.Name)
		sb.WriteString(",")
		sb.WriteString(sp.Value)
		sb.WriteString("}")
	}
	return sb.String()
}

func (a *artifact) String() string {
	sb := strings.Builder{}
	sb.WriteString("artifact{")
	sb.WriteString("props:")
	sb.WriteString(a.props.String())
	sb.WriteString("}")
	return sb.String()
}

func (pp *PropertiesTemplate) ensureInitialized() {
	if pp.constProps == nil {
		pp.constProps = make(map[stringProperty]bool)
		pp.additionalProps = make(map[string]bool)
	}
}

func (pp *PropertiesTemplate) AddConstantProperty(name string, value string) {
	pp.ensureInitialized()
	pair := stringProperty{Name: name, Value: value}
	pp.constProps[pair] = true
}

func (pp *PropertiesTemplate) Has(name string, value string) bool {
	if pp.constProps == nil {
		return false
	}
	return pp.constProps[stringProperty{Name: name, Value: value}]
}

func (pp *PropertiesTemplate) Contains(other *PropertiesTemplate) bool {
	for pair, _ := range other.constProps {
		if !pp.Has(pair.Name, pair.Value) {
			return false
		}
	}
	return true
}
