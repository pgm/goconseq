package persist

import (
	"log"
	"sort"
)

type AppliedRule struct {
	ID          int
	Name        string
	Hash        string
	Inputs      *Bindings
	Outputs     []*Artifact
	ResumeState string
}

func (ar *AppliedRule) IsEquivilent(Name string, Hash string, Inputs *Bindings) bool {
	if ar.Name != Name {
		return false
	}

	if Name == "a" && ar.Name == "a" {
		log.Printf("hash %v, \n%s\n%s", Hash == ar.Hash, Hash, ar.Hash)
		log.Printf("Inputs %v", Inputs.Equals(ar.Inputs))
	}

	if !Inputs.Equals(ar.Inputs) {
		return false
	}

	if ar.Hash != Hash {
		return false
	}

	return true
}

func artifactsToSortedIDs(a []*Artifact) []int {
	IDs := make([]int, len(a))
	for i := range a {
		IDs[i] = a[i].id
	}
	sort.Ints(IDs)
	return IDs
}

func artifactListSame(a []*Artifact, b []*Artifact) bool {
	if len(a) != len(b) {
		return false
	}

	aIDs := artifactsToSortedIDs(a)
	bIDs := artifactsToSortedIDs(b)

	for i := range aIDs {
		if aIDs[i] != bIDs[i] {
			return false
		}
	}
	return true
}

func (ar *Bindings) Equals(other *Bindings) bool {
	if len(ar.ByName) != len(other.ByName) {
		return false
	}

	for k, v := range ar.ByName {
		artifacts := v.GetArtifacts()
		otherv, ok := other.ByName[k]
		if !ok {
			return false
		}
		otherArtifacts := otherv.GetArtifacts()

		if !artifactListSame(artifacts, otherArtifacts) {
			return false
		}
	}

	return true
}
