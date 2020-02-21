package persist

import (
	"sort"
	"strconv"
	"strings"
)

type BindingValue interface {
	GetArtifacts() []*Artifact
	Hash() string
}

type Bindings struct {
	ByName map[string]BindingValue
}

func NewBindings() *Bindings {
	return &Bindings{ByName: make(map[string]BindingValue)}
}

var EmptyBinding = NewBindings()

type MultipleArtifacts struct {
	artifacts []*Artifact
}

func (m *MultipleArtifacts) GetArtifacts() []*Artifact {
	return m.artifacts
}

func (m *MultipleArtifacts) Hash() string {
	sb := strings.Builder{}
	sb.WriteString("(")
	for _, artifact := range m.artifacts {
		sb.WriteString(strconv.Itoa(artifact.id))
		sb.WriteString(",")
	}
	sb.WriteString(")")

	return sb.String()
}

type SingleArtifact struct {
	MultipleArtifacts
}

func (b *Bindings) Hash() string {
	keys := make([]string, len(b.ByName))
	i := 0
	for name := range b.ByName {
		keys[i] = name
		i++
	}
	sort.Strings(keys)

	sb := strings.Builder{}
	sb.WriteString("(")
	for _, name := range keys {
		sb.WriteString(escapeStr(name))
		sb.WriteString(":")
		value := b.ByName[name]
		sb.WriteString(value.Hash())
		sb.WriteString(",")
	}
	sb.WriteString(")")
	return sb.String()
}

func (b *Bindings) AddArtifacts(name string, artifacts []*Artifact) {
	b.ByName[name] = &MultipleArtifacts{artifacts}
}

func (b *Bindings) AddArtifact(name string, artifact *Artifact) {
	b.ByName[name] = &SingleArtifact{MultipleArtifacts: MultipleArtifacts{[]*Artifact{artifact}}}
}

func (b *Bindings) Transform(transform func(artifact *Artifact) *Artifact) *Bindings {
	nb := NewBindings()
	for name, value := range b.ByName {
		s, ok := value.(*SingleArtifact)
		if ok {
			nb.AddArtifact(name, transform(s.artifacts[0]))
		} else {
			sArtifacts := value.GetArtifacts()
			artifacts := make([]*Artifact, len(sArtifacts))
			for i, sArtifact := range sArtifacts {
				artifacts[i] = transform(sArtifact)
			}
			nb.AddArtifacts(name, artifacts)
		}
	}
	return nb
}
