package persist

import (
	"fmt"
	"log"
	"os"
	"path"
)

// stored types: Artifacts, AppliedRules
// TODO: Make db threadsafe
const InitialStep = 0

type File struct {
	FileID     int
	LocalPath  string
	GlobalPath string
	SHA256     string
}

type DB struct {
	// todo: make DB methods threadsafe

	nextID              int
	nextAppliedRuleID   int
	currentArtifacts    map[int]*Artifact    // the in-memory set of all artifacts which resulted from the current run
	currentAppliedRules map[int]*AppliedRule // the set of all appliedRules which participated in the current run
	stateDir            string
	writer              *OpLogWriter

	artifactHistoryByHash map[string]*Artifact // all artifacts ever generated
	artifactHistoryByID   map[int]*Artifact    // all artifacts ever generated
	// appliedRuleHistoryByHash map[string]*AppliedRule // all artifacts ever generated
	appliedRuleHistoryByID map[int]*AppliedRule // all artifacts ever generated
	files                  map[int]*File
}

type DBOp interface {
	Update(db *DB)

	GetType() string
}

func NewDB(stateDir string) *DB {
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		err = os.MkdirAll(stateDir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	db := &DB{
		nextID:                 1,
		currentArtifacts:       make(map[int]*Artifact),
		artifactHistoryByID:    make(map[int]*Artifact),
		artifactHistoryByHash:  make(map[string]*Artifact),
		currentAppliedRules:    make(map[int]*AppliedRule),
		appliedRuleHistoryByID: make(map[int]*AppliedRule),
		files:                  make(map[int]*File),
		// appliedRuleHistoryByHash: make(map[string]*AppliedRule),
		stateDir: stateDir}

	logPath := path.Join(stateDir, "db.journal")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		db.loadFromJournal(logPath)
	}

	writer, err := OpenLogWriter(logPath)
	if err != nil {
		panic(err)
	}

	db.writer = writer
	return db
}

func (db *DB) DisableUpdates() {
	db.writer.disableWrites = true
}

func (db *DB) loadFromJournal(filename string) {
	reader, err := OpenLogReader(filename)
	if err != nil {
		panic(err)
	}
	for {
		ops, err := reader.ReadTransaction()
		if err != nil {
			log.Printf("Read log up to %v", err)
			break
		}
		for _, op := range ops {
			op.Update(db)
		}
	}
	reader.Close()
}

func (db *DB) Close() {
	db.writer.Close()
}

func (db *DB) GetNextApplicationID() int {
	ID := db.nextAppliedRuleID
	db.writer.WriteSetNextIDs(db.nextID, db.nextAppliedRuleID+1).Update(db)
	db.writer.Commit()
	return ID
}

func (db *DB) GetWorkDir(appliedRuleID int) string {
	return path.Join(db.stateDir,
		fmt.Sprintf("r%d", appliedRuleID))
}

func (db *DB) GetHackCount() int {
	return len(db.currentAppliedRules)
}

// all _read_ operations do not return errors because they only use memory. All _write_ operations return an error
func (db *DB) FindAppliedRule(Name string, Hash string, Inputs *Bindings) *AppliedRule {
	log.Printf("FindAppliedRule %s among %d", Name, len(db.currentAppliedRules))
	for _, appliedRule := range db.currentAppliedRules {
		//		fmt.Printf("%v %v %v\n", Name == appliedRule.Name, Hash == appliedRule.Hash, Inputs.Equals(appliedRule.Inputs))
		//		fmt.Printf("%s == %s -> %v, %s == %s -> %v, eqiv %v\n", Name, appliedRule.Name, Name == appliedRule.Name, Hash, appliedRule.Hash, Hash == appliedRule.Hash, Inputs.Equals(appliedRule.Inputs))

		if appliedRule.IsEquivilent(Name, Hash, Inputs) {
			return appliedRule
		}
	}
	return nil
}

func (db *DB) GetAppliedRule(id int) *AppliedRule {
	return db.appliedRuleHistoryByID[id]
}

// writes AppliedRule to history _and_ adds as a current rule application
func (db *DB) PersistAppliedRule(ID int, Name string, Hash string, Inputs *Bindings, ResumeState string) (*AppliedRule, error) {
	appliedRule := &AppliedRule{ID: ID, Name: Name, Inputs: Inputs, ResumeState: ResumeState, Hash: Hash}

	db.writer.WriteSetAppliedRule(appliedRule).Update(db)
	db.writer.Commit()

	return appliedRule, nil
}

func (db *DB) AddAppliedRuleToCurrent(ID int) {
	// this is for promoting a past applied rule execution from the history to add it to the current session

	appliedRule := db.appliedRuleHistoryByID[ID]
	db.currentAppliedRules[appliedRule.ID] = appliedRule

	// if this rule is complete, add all the artifacts to the current session as well
	if appliedRule.Outputs != nil {
		for _, output := range appliedRule.Outputs {
			if _, exists := db.currentArtifacts[output.id]; exists {
				panic(fmt.Sprintf("Cannot record completion of applied rule because artifact is already in session: %v", output.String()))
			}
			db.currentArtifacts[output.id] = output
		}
	}
}

func (db *DB) DumpArtifacts() {
	for i, a := range db.currentArtifacts {
		fmt.Printf("artifact %d: %v", i, a.Properties.ToStrMap())
	}
}

func (db *DB) UpdateAppliedRuleComplete(ID int, Outputs []*Artifact) error {
	appliedRule := *db.appliedRuleHistoryByID[ID]
	appliedRule.Outputs = Outputs
	appliedRule.ResumeState = ""

	db.writer.WriteSetAppliedRule(&appliedRule).Update(db)
	db.writer.Commit()

	for _, output := range Outputs {
		log.Printf("adding output.id=%d", output.id)
		if _, exists := db.currentArtifacts[output.id]; exists {
			return fmt.Errorf("Cannot record completion of applied rule because artifact is already in session: %v", output.String())
		}
		db.currentArtifacts[output.id] = output
	}
	// if _, exists := db.currentAppliedRules[appliedRule.ID]; !exists {
	// 	panic("Application should already exist")
	// }
	db.currentAppliedRules[appliedRule.ID] = &appliedRule

	return nil
}

func (db *DB) PersistArtifact(Properties *ArtifactProperties) (*Artifact, error) {
	id := db.nextID
	artifact := &Artifact{id: id, Properties: Properties}

	db.writer.WriteSetNextIDs(db.nextID+1, db.nextAppliedRuleID).Update(db)
	db.writer.WriteSetArtifact(artifact).Update(db)
	db.writer.Commit()

	return artifact, nil
}

// func (db *DB) FindExactArtifact(Properties map[string]string) (*Artifact, error) {
// }

// Search among the current artifacts to find artifact with the properties/values
func (db *DB) FindArtifacts(Properties map[string]string) []*Artifact {
	results := make([]*Artifact, 0, 10)
	for _, artifact := range db.currentArtifacts {
		if artifact.HasProperties(Properties) {
			results = append(results, artifact)
		}
	}
	return results
}

func (db *DB) FindAllAppliedRules() []*AppliedRule {
	result := make([]*AppliedRule, 0, len(db.currentAppliedRules))
	for _, appliedRule := range db.currentAppliedRules {
		result = append(result, appliedRule)
	}
	return result
}

func (db *DB) AddFileGlobalPath(localPath string, globalPath string, sha256 string) *File {
	fileID := db.nextID
	file := &File{FileID: fileID, LocalPath: localPath, GlobalPath: globalPath, SHA256: sha256}
	db.files[fileID] = file

	db.writer.WriteSetNextIDs(db.nextID+1, db.nextAppliedRuleID).Update(db)
	db.writer.WriteSetFile(file)
	db.writer.Commit()

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

	db.writer.WriteSetFile(file)
	db.writer.Commit()

	return file
}

func (db *DB) GetAppliedRuleFromHistory(name string, hash string, inputs *Bindings) *AppliedRule {
	log.Printf("appliedRuleHistoryByID %s among %d", name, len(db.appliedRuleHistoryByID))
	var found *AppliedRule
	for _, appliedRule := range db.appliedRuleHistoryByID {
		if appliedRule.IsEquivilent(name, hash, inputs) {
			if found == nil {
				found = appliedRule
			} else {
				panic("Too many matches")
			}
		}
	}
	return found
}

func (db *DB) GetArtifactFromHistory(props *ArtifactProperties) *Artifact {
	var found *Artifact
	propsHash := props.Hash()
	for _, artifact := range db.artifactHistoryByID {
		if artifact.Properties.Hash() == propsHash {
			if found == nil {
				found = artifact
			} else {
				panic("Too many matches")
			}
		}
	}
	return found
}

func (db *DB) DeleteAppliedRule(ID int) error {
	app := db.appliedRuleHistoryByID[ID]
	appsToDelete := []*AppliedRule{app}
	appsToDelete = append(appsToDelete, db.FindApplicationsDownstreamOfApplication(app.ID)...)

	for _, app = range appsToDelete {
		for _, artifact := range app.Outputs {
			db.writer.WriteDeleteArtifact(artifact.id).Update(db)
		}
		db.writer.WriteDeleteAppliedRule(app.ID)
	}
	db.writer.Commit()
	return nil
}

func (db *DB) FindRuleApplicationsWithInput(artifact *Artifact) []*AppliedRule {
	appliedRules := make([]*AppliedRule, 0, 10)
outerLoop:
	for _, appliedRule := range db.currentAppliedRules {
		for _, value := range appliedRule.Inputs.ByName {
			for _, a := range value.GetArtifacts() {
				if a.id == artifact.id {
					appliedRules = append(appliedRules, appliedRule)
					continue outerLoop
				}
			}
		}
	}
	return appliedRules
}

func (db *DB) FindApplicationsDownstreamOfArtifact(artifact *Artifact) []*AppliedRule {
	result := make([]*AppliedRule, 0)

	applications := db.FindRuleApplicationsWithInput(artifact)
	for _, application := range applications {
		result = append(result, application)
		result = append(result, db.FindApplicationsDownstreamOfApplication(application.ID)...)
	}
	return result
}

func (db *DB) FindApplicationsDownstreamOfApplication(appliedRuleID int) []*AppliedRule {
	appliedRule, exists := db.currentAppliedRules[appliedRuleID]
	if !exists {
		panic("Looked up missing appliedRuleID")
	}
	result := make([]*AppliedRule, 0)
	for _, output := range appliedRule.Outputs {
		result = append(result, db.FindApplicationsDownstreamOfArtifact(output)...)
	}

	return result
}

func (db *DB) AddFileOrFind(localPath, sha256 string) int {
	for _, file := range db.files {
		if file.SHA256 == sha256 {
			return file.FileID
		}
	}

	f := db.AddFileGlobalPath(localPath, "", sha256)
	return f.FileID
}

// func (db *DB) FindAppliedRulesByName(name string) (*AppliedRule, error) {
// }

// func (db *DB) FindAppliedRulesByInput(id int) ([]*AppliedRule, error) {

// }

// func (db *DB) FindAppliedRulesByOutput(id int) ([]*AppliedRule, error) {

// }
