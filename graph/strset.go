package graph

import (
	"sort"
	"strings"
)

type strSet struct {
	set map[string]bool
}

func newStrSet() *strSet {
	return &strSet{make(map[string]bool)}
}

func (ss *strSet) String() string {
	values := make([]string, 0, len(ss.set))
	for k := range ss.set {
		values = append(values, k)
	}
	sort.Strings(values)
	return "{" + strings.Join(values, " ") + "}"
}

func (ss *strSet) Add(value string) bool {
	if ss.set == nil {
		ss.set = make(map[string]bool)
	}
	exists := ss.set[value]
	ss.set[value] = true
	return !exists
}

func (ss *strSet) ForEach(fn func(string)) {
	if ss.set == nil {
		return
	}
	for value := range ss.set {
		fn(value)
	}
}

func (ss *strSet) Length() int {
	if ss.set == nil {
		return 0
	}
	return len(ss.set)
}

func (ss *strSet) Remove(value string) {
	if ss.set == nil {
		return
	}
	delete(ss.set, value)
}

func (ss *strSet) Has(value string) bool {
	if ss.set == nil {
		return false
	}
	return ss.set[value]
}
