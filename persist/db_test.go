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

func TestFindRuleApplicationsWithInput(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	assert.Nil(t, err)
	defer os.RemoveAll(stateDir)

	dir := path.Join(stateDir, "db")
	db := NewDB(dir)

	joe, _ := db.PersistArtifact(&ArtifactProperties{Strings: map[string]string{"name": "joe"}})
	mary, _ := db.PersistArtifact(&ArtifactProperties{Strings: map[string]string{"name": "mary"}})
	steve, _ := db.PersistArtifact(&ArtifactProperties{Strings: map[string]string{"name": "steve"}})

	saveAppWithArtifact := func(artifact *Artifact) *AppliedRule {
		appID := db.GetNextApplicationID()
		app, err := db.PersistAppliedRule(appID, "gen_"+artifact.Properties.Strings["name"], "hash", NewBindings(), "")
		assert.Nil(t, err)
		err = db.UpdateAppliedRuleComplete(app.ID, []*Artifact{artifact})
		assert.Nil(t, err)
		return app
	}

	saveMergedApp := func() *AppliedRule {
		appID := db.GetNextApplicationID()
		bindings := NewBindings()
		bindings.AddArtifact("person1", joe)
		bindings.AddArtifact("person2", mary)
		app, err := db.PersistAppliedRule(appID, "merge", "hash", bindings, "")
		assert.Nil(t, err)
		err = db.UpdateAppliedRuleComplete(app.ID, []*Artifact{steve})
		assert.Nil(t, err)
		return app
	}

	app1 := saveAppWithArtifact(joe)
	app2 := saveAppWithArtifact(mary)
	app3 := saveMergedApp()

	// make sure find apps via input works for all three artifacts
	assert.Empty(t, db.FindRuleApplicationsWithInput(steve))

	apps := db.FindRuleApplicationsWithInput(joe)
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, app3.ID, apps[0].ID)

	apps = db.FindRuleApplicationsWithInput(mary)
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, app3.ID, apps[0].ID)

	// make sure find downstream of apps works
	apps = db.FindApplicationsDownstreamOfApplication(app1.ID)
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, app3.ID, apps[0].ID)

	apps = db.FindApplicationsDownstreamOfApplication(app2.ID)
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, app3.ID, apps[0].ID)

	apps = db.FindApplicationsDownstreamOfApplication(app3.ID)
	assert.Equal(t, 0, len(apps))

	// make sure find downstream of artifact works
	apps = db.FindApplicationsDownstreamOfArtifact(joe)
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, app3.ID, apps[0].ID)

	apps = db.FindApplicationsDownstreamOfArtifact(mary)
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, app3.ID, apps[0].ID)

	apps = db.FindApplicationsDownstreamOfArtifact(steve)
	assert.Equal(t, 0, len(apps))
}

func TestSimpleQuery(t *testing.T) {
	stateDir, err := ioutil.TempDir("", t.Name())
	assert.Nil(t, err)
	defer os.RemoveAll(stateDir)

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
