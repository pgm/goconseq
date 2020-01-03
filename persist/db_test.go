package persist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinedQuery(t *testing.T) {
	db := NewDB()

	joePerson, _ := db.PersistArtifact(InitialStep, &ArtifactProperties{Strings: map[string]string{"type": "person", "name": "joe", "select": "a"}})
	joeAddress, _ := db.PersistArtifact(InitialStep, &ArtifactProperties{Strings: map[string]string{"type": "address", "name": "joe"}})
	stevePerson, _ := db.PersistArtifact(InitialStep, &ArtifactProperties{Strings: map[string]string{"type": "person", "name": "steve", "select": "b"}})
	steveAddress, _ := db.PersistArtifact(InitialStep, &ArtifactProperties{Strings: map[string]string{"type": "address", "name": "steve"}})

	fetchAndVerify := func(personID int, addressID int, propValue string) {
		query := &Query{
			forEach: []*QueryBinding{
				&QueryBinding{
					bindingVariable: "person",
					constantConstraints: map[string]string{
						"type":   "person",
						"select": propValue},
					placeholderAssignments: []StringPair{StringPair{"name", "NAME"}}},
				&QueryBinding{
					bindingVariable: "address",
					constantConstraints: map[string]string{
						"type": "address"},
					placeholderConstraints: []StringPair{StringPair{"name", "NAME"}}}}}
		bindings := ExecuteQuery(db, query)
		assert.Equal(t, 1, len(bindings))
		artifacts := bindings[0].ByName["person"].GetArtifacts()
		assert.Equal(t, 1, len(artifacts))
		assert.Equal(t, personID, artifacts[0].id)
		artifacts = bindings[0].ByName["address"].GetArtifacts()
		assert.Equal(t, 1, len(artifacts))
		assert.Equal(t, addressID, artifacts[0].id)
	}

	fetchAndVerify(joePerson.id, joeAddress.id, "a")
	fetchAndVerify(stevePerson.id, steveAddress.id, "b")
}

func TestSimpleQuery(t *testing.T) {
	db := NewDB()

	a1, _ := db.PersistArtifact(InitialStep, &ArtifactProperties{Strings: map[string]string{"prop": "true", "common": "shared"}})
	a2, _ := db.PersistArtifact(InitialStep, &ArtifactProperties{Strings: map[string]string{"prop": "false", "common": "shared"}})

	makeQuery := func(propName string, propValue string) *Query {
		return &Query{
			forEach: []*QueryBinding{
				&QueryBinding{
					bindingVariable: "var",
					constantConstraints: map[string]string{
						propName: propValue}}}}
	}

	fetchAndVerify := func(expectedID int, propValue string) {
		query := makeQuery("prop", propValue)
		bindings := ExecuteQuery(db, query)
		assert.Equal(t, 1, len(bindings))
		artifacts := bindings[0].ByName["var"].GetArtifacts()
		assert.Equal(t, 1, len(artifacts))
		assert.Equal(t, expectedID, artifacts[0].id)
	}

	// queries that select a single artifact
	fetchAndVerify(a2.id, "false")
	fetchAndVerify(a1.id, "true")

	// and now try a query that should return nothing
	empty := ExecuteQuery(db, makeQuery("prop", "other"))
	assert.Equal(t, 0, len(empty))

	// and a query that selects all artifacts
	all := ExecuteQuery(db, makeQuery("common", "shared"))
	assert.Equal(t, 2, len(all))
}
