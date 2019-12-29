package persist

type Artifact struct {
	id         int
	ProducedBy int
	Properties map[string]string
}

func (a *Artifact) HasProperties(props map[string]string) bool {
	for k, v := range props {
		if a.Properties[k] != v {
			return false
		}
	}
	return true
}

func (a *Artifact) PropertiesEqual(other *Artifact) bool {
	if len(a.Properties) != len(other.Properties) {
		return false
	}

	return a.HasProperties(other.Properties)
}
