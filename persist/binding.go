package persist

type BindingValue interface {
	GetArtifacts() []*Artifact
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

type SingleArtifact struct {
	MultipleArtifacts
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
		if !ok {
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
