package persist

type BindingValue interface {
	GetArtifacts() []*Artifact
}

type Bindings struct {
	ByName map[string]BindingValue
}

type MultipleArtifacts struct {
	artifacts []*Artifact
}

func (m *MultipleArtifacts) GetArtifacts() []*Artifact {
	return m.artifacts
}

type SingleArtifact struct {
	MultipleArtifacts
}

func (b *Bindings) AddArtifacts(name string, artifacts []*Artifact) {
	b.ByName[name] = &MultipleArtifacts{artifacts}
}

func (b *Bindings) AddArtifact(name string, artifact *Artifact) {
	b.ByName[name] = &SingleArtifact{MultipleArtifacts: MultipleArtifacts{[]*Artifact{artifact}}}
}
