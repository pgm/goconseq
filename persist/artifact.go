package persist

type ArtifactProperties struct {
	Strings map[string]string
	Files   map[string]int
}

type Artifact struct {
	id         int
	ProducedBy int
	Properties ArtifactProperties
}

func (a *Artifact) HasProperties(props map[string]string) bool {
	for k, v := range props {
		if a.Properties.Strings[k] != v {
			return false
		}
	}
	return true
}

func (a *Artifact) PropertiesEqual(other *Artifact) bool {
	if len(a.Properties.Strings) != len(other.Properties.Strings) {
		return false
	}

	return a.HasProperties(other.Properties.Strings)
}
