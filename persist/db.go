package persist

import (
	"github.com/pgm/goconseq/graph"
)

// stored types: Artifacts, AppliedRules
// TODO: Make db threadsafe
const InitialStep = 0

type File struct {
	FileID     int
	LocalPath  string
	GlobalPath string
}

type DB struct {
	nextArtifactID    int
	nextAppliedRuleID int
	nextFileID        int
	artifacts         map[int]*Artifact
	appliedRules      map[int]*AppliedRule
	files             map[int]*File
}

func NewDB() *DB {
	return &DB{artifacts: make(map[int]*Artifact), appliedRules: make(map[int]*AppliedRule)}
}

func (db *DB) GetNextApplicationID() int {
	ID := db.nextAppliedRuleID
	db.nextAppliedRuleID++
	return ID
}

// all _read_ operations do not return errors because they only use memory. All _write_ operations return an error
func (db *DB) FindAppliedRule(Name string, Inputs *Bindings) *AppliedRule {
	for _, appliedRule := range db.appliedRules {
		if appliedRule.IsEquivilent(Name, Inputs) {
			return appliedRule
		}
	}
	return nil
}

func (db *DB) GetAppliedRule(id int) *AppliedRule {
	return db.appliedRules[id]
}

func (db *DB) PersistAppliedRule(ID int, Name string, Inputs *Bindings, ResumeState string) (*AppliedRule, error) {
	appliedRule := &AppliedRule{id: ID, Name: Name, Inputs: Inputs, ResumeState: ResumeState}
	db.appliedRules[ID] = appliedRule

	return appliedRule, nil
}

func (db *DB) UpdateAppliedRuleComplete(ID int, Outputs []*Artifact) error {
	db.appliedRules[ID].Outputs = Outputs
	db.appliedRules[ID].ResumeState = ""
	return nil
}

func (db *DB) DeleteAppliedRule(ID int) error {
	delete(db.appliedRules, ID)
	return nil
}

// func (db *DB) DeleteArtifact(ID int) error {
// }

func (db *DB) PersistArtifact(ProducedBy int, Properties *ArtifactProperties) (*Artifact, error) {
	id := db.nextArtifactID
	db.nextArtifactID++
	artifact := &Artifact{id: id, ProducedBy: ProducedBy, Properties: *Properties}
	db.artifacts[id] = artifact

	return artifact, nil
}

// func (db *DB) FindExactArtifact(Properties map[string]string) (*Artifact, error) {
// }

func (db *DB) FindArtifacts(Properties map[string]string) []*Artifact {
	results := make([]*Artifact, 0, 10)
	for _, artifact := range db.artifacts {
		if artifact.HasProperties(Properties) {
			results = append(results, artifact)
		}
	}
	return results
}

func (db *DB) AddFileGlobalPath(localPath string, globalPath string) *File {
	fileID := db.nextFileID
	db.nextFileID++
	file := &File{FileID: fileID, LocalPath: localPath, GlobalPath: globalPath}
	db.files[fileID] = file
	return file
}

func (db *DB) GetFile(fileID int) *File {
	return db.files[fileID]
}

func (db *DB) UpdateFile(fileID int, localPath string, globalPath string) *File {
	// never mutate, make a copy
	origFile := db.files[fileID]
	if localPath == "" {
		localPath = origFile.LocalPath
	}
	if globalPath == "" {
		globalPath = origFile.GlobalPath
	}
	file := &File{FileID: fileID, GlobalPath: globalPath, LocalPath: localPath}
	db.files[fileID] = file
	return file
}

// func (db *DB) FindAppliedRulesByName(name string) (*AppliedRule, error) {
// }

// func (db *DB) FindAppliedRulesByInput(id int) ([]*AppliedRule, error) {

// }

// func (db *DB) FindAppliedRulesByOutput(id int) ([]*AppliedRule, error) {

// }

type StringPair struct {
	first  string
	second string
}

type QueryBinding struct {
	// the variable to assign the artifact returned to
	bindingVariable string
	// the static constraints to use when querying
	constantConstraints map[string]string
	// the variable constraints to use when querying. Each of these will reference a field from a prior variable definition
	placeholderConstraints []StringPair
	placeholderAssignments []StringPair
}

// type BindingProperty struct {
// 	bindingVariable string
// 	name            string
// }

type Query struct {
	forEach []*QueryBinding
	forAll  []*QueryBinding
}

func (q *Query) GetProps() []*graph.PropertiesTemplate {
	result := make([]*graph.PropertiesTemplate, len(q.forEach))
	for i, qb := range q.forEach {
		pp := graph.PropertiesTemplate{}
		for name, value := range qb.constantConstraints {
			pp.AddConstantProperty(name, value)
		}
		result[i] = &pp
	}
	return result
}

func mergeConstraints(original map[string]string,
	substitutions []StringPair,
	placeholders map[string]string) map[string]string {

	merged := make(map[string]string)
	for k, v := range original {
		merged[k] = v
	}
	for i := range substitutions {
		merged[substitutions[i].first] = placeholders[substitutions[i].second]
	}
	return merged
}

func copyStrMap(a map[string]string) map[string]string {
	b := make(map[string]string, len(a))
	for k, v := range a {
		b[k] = v
	}
	return b
}

func _executeQuery(db *DB, origPlaceholders map[string]string, forEachList []*QueryBinding) []*Bindings {
	forEach := forEachList[0]
	restForEach := forEachList[1:]

	constraints := mergeConstraints(forEach.constantConstraints, forEach.placeholderConstraints, origPlaceholders)
	artifacts := db.FindArtifacts(constraints)
	if len(artifacts) == 0 {
		return nil
	}

	if len(restForEach) == 0 {
		// base case: return the bindings
		records := make([]*Bindings, len(artifacts))
		for i := range artifacts {
			binding := &Bindings{ByName: make(map[string]BindingValue)}
			binding.AddArtifact(forEach.bindingVariable, artifacts[i])
			records[i] = binding
		}
		return records
	}

	// recursive case: execute _executeQuery on the remainder of forEaches
	combinedRecords := make([]*Bindings, 0, len(artifacts))
	for _, artifact := range artifacts {
		// before invoking next query, record any placeholders based on the current artifact
		placeholders := copyStrMap(origPlaceholders)
		for _, assignment := range forEach.placeholderAssignments {
			placeholders[assignment.second] = artifact.Properties.Strings[assignment.first]
		}
		records := _executeQuery(db, placeholders, restForEach)
		for _, record := range records {
			binding := &Bindings{ByName: make(map[string]BindingValue)}
			binding.AddArtifact(forEach.bindingVariable, artifact)
			for k, v := range record.ByName {
				binding.ByName[k] = v
			}
			combinedRecords = append(combinedRecords, binding)
		}
	}
	return combinedRecords
}

func ExecuteQuery(db *DB, query *Query) []*Bindings {
	// resolve all forEaches before doing any forAlls
	placeholders := make(map[string]string)
	if len(query.forAll) != 0 {
		panic("forall not implemented")
	}
	return _executeQuery(db, placeholders, query.forEach)
}

func QueryFromMaps(bindMap map[string]map[string]string) *Query {
	var query Query

	for name, template := range bindMap {
		binding := &QueryBinding{bindingVariable: name,
			constantConstraints: template}
		// bindingVariable string
		// // the static constraints to use when querying
		// constantConstraints map[string]string
		// // the variable constraints to use when querying. Each of these will reference a field from a prior variable definition
		// placeholderConstraints []StringPair
		// placeholderAssignments []StringPair

		// for k, v := template {
		// }
		query.forEach = append(query.forEach, binding)
	}

	return &query
}
