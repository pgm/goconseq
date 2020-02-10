package persist

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddFiles(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	assert.Nil(t, err)
	defer os.RemoveAll(stateDir)

	fn := path.Join(stateDir, "sample")
	f, err := os.Create(fn)
	assert.Nil(t, err)
	f.WriteString("x")
	f.Close()

	dir := path.Join(stateDir, "db")
	db := NewDB(dir)

	fileID := db.AddFileOrFind(fn, "abc")
	assert.True(t, fileID > 0)
	db.Close()

	db = NewDB(dir)
	ff := db.files[fileID]
	assert.Equal(t, fileID, ff.FileID)
	assert.Equal(t, fn, ff.LocalPath)
	assert.Equal(t, "abc", ff.SHA256)
	db.Close()
}

func TestJoinedQuery(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	assert.Nil(t, err)
	defer os.RemoveAll(stateDir)

	dir := path.Join(stateDir, "db")
	db := NewDB(dir)

	joePerson, _ := db.PersistArtifact(&ArtifactProperties{Strings: map[string]string{"type": "person", "name": "joe", "select": "a"}})
	joeAddress, _ := db.PersistArtifact(&ArtifactProperties{Strings: map[string]string{"type": "address", "name": "joe"}})
	stevePerson, _ := db.PersistArtifact(&ArtifactProperties{Strings: map[string]string{"type": "person", "name": "steve", "select": "b"}})
	steveAddress, _ := db.PersistArtifact(&ArtifactProperties{Strings: map[string]string{"type": "address", "name": "steve"}})
	appID := db.GetNextApplicationID()

	app, err := db.PersistAppliedRule(appID, "init", "hash", NewBindings(), "")
	db.UpdateAppliedRuleComplete(app.ID, []*Artifact{joePerson, joeAddress, stevePerson, steveAddress})
	assert.Nil(t, err)

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

	db.Close()

	// verify we can reconstruct the artifacts and applications after reopening the DB
	db = NewDB(dir)
	defer db.Close()

	// compare the number of artifacts
	assert.Equal(t, 4, len(db.artifactHistoryByID))
	assert.Equal(t, 4, len(db.artifactHistoryByHash))

	// compare a single artifact's properties
	newJoePerson := db.artifactHistoryByID[joePerson.id]
	assert.NotEqual(t, newJoePerson, joePerson)
	assert.Equal(t, newJoePerson.Properties.Hash(), joePerson.Properties.Hash())
}

func TestSimpleQuery(t *testing.T) {
	stateDir, err := ioutil.TempDir("", "TestSimpleQuery")
	assert.Nil(t, err)

	dir := path.Join(stateDir, "db")
	db := NewDB(dir)

	a1, _ := db.PersistArtifact(&ArtifactProperties{Strings: map[string]string{"prop": "true", "common": "shared"}})
	a2, _ := db.PersistArtifact(&ArtifactProperties{Strings: map[string]string{"prop": "false", "common": "shared"}})
	appID := db.GetNextApplicationID()
	app, err := db.PersistAppliedRule(appID, "init", "hash", NewBindings(), "")
	db.UpdateAppliedRuleComplete(app.ID, []*Artifact{a1, a2})
	assert.Nil(t, err)

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

	allChecks := func() {
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

	allChecks()
	db.Close()

	// verify everything still works after we close and reopen the db
	db = NewDB(dir)
	defer db.Close()

	assert.Equal(t, 2, len(db.artifactHistoryByID))
	assert.Equal(t, 2, len(db.artifactHistoryByHash))
}
