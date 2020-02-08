package persist

import (
	"log"

	"github.com/pgm/goconseq/graph"
)

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

func (q *QueryBinding) AsDict() map[string]interface{} {
	log.Printf("Warning: QueryBinding is incomplete")
	return map[string]interface{}{
		"bindingVariable":     q.bindingVariable,
		"constantConstraints": q.constantConstraints,
	}
}

func queryBindingSliceAsDict(v []*QueryBinding) []interface{} {
	nv := make([]interface{}, len(v))
	for i := range v {
		nv[i] = v[i].AsDict()
	}
	return nv
}

func (q *Query) AsDict() map[string]interface{} {
	if q == nil {
		return nil
	}

	return map[string]interface{}{
		"forEach": queryBindingSliceAsDict(q.forEach),
		"forAll":  queryBindingSliceAsDict(q.forAll)}
}

func (q *Query) IsEmpty() bool {
	return len(q.forEach) == 0 && len(q.forAll) == 0
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
	if len(query.forEach) == 0 {
		panic("need at least one foreach")
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
