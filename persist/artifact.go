package persist

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type ArtifactProperties struct {
	Strings map[string]string
	Files   map[string]int
}

type Artifact struct {
	id         int
	Properties *ArtifactProperties
}

func (ap *ArtifactProperties) ToStrMap(fmtFile func(fileID int) string) map[string]string {
	if fmtFile == nil {
		fmtFile = func(fileID int) string {
			return fmt.Sprintf("<file %d>", fileID)
		}
	}
	result := make(map[string]string)
	for k, v := range ap.Strings {
		result[k] = v
	}
	for k, v := range ap.Files {
		result[k] = fmtFile(v)
	}
	return result
}

func (ap *ArtifactProperties) String() string {
	m := make(map[string]interface{})
	for k, v := range ap.Strings {
		m[k] = v
	}
	for k, i := range ap.Files {
		m[k] = fmt.Sprintf("<fileref %d>", i)
	}

	s, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return string(s)
}

func (a *Artifact) String() string {
	return fmt.Sprintf("<artifact %d %s>", a.id, a.Properties.String())
}

func escapeStr(s string) string {
	return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
}

// compute a hash of the properties which we can use to find equivilent artifacts
func (ap *ArtifactProperties) Hash() string {
	stringsKeys := make([]string, len(ap.Strings))
	i := 0
	for name := range ap.Strings {
		stringsKeys[i] = name
		i++
	}
	sort.Strings(stringsKeys)

	filesKeys := make([]string, len(ap.Files))
	i = 0
	for name := range ap.Files {
		filesKeys[i] = name
		i++
	}
	sort.Strings(filesKeys)

	b := strings.Builder{}
	for _, name := range stringsKeys {
		value := ap.Strings[name]
		b.WriteString("S")
		b.WriteString(escapeStr(name))
		b.WriteString(":")
		b.WriteString(escapeStr(value))
		b.WriteString(",")
	}
	for _, name := range filesKeys {
		fileID := ap.Files[name]
		b.WriteString("F")
		b.WriteString(escapeStr(name))
		b.WriteString(":")
		b.WriteString(strconv.Itoa(fileID))
		b.WriteString(",")
	}
	return b.String()
}

func NewArtifactProperties() *ArtifactProperties {
	return &ArtifactProperties{Strings: map[string]string{}, Files: map[string]int{}}
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
