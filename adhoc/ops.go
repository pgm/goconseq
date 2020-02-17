package adhoc

type KVPairs map[string]string

func GroupBy(pairsList []KVPairs, field string) map[string][]KVPairs {
	result := make(map[string][]KVPairs)
	for _, pairs := range pairsList {
		fieldValue := pairs[field]
		result[fieldValue] = append(result[fieldValue], pairs)
	}
	return result
}

func SelectSingle(pairs KVPairs, fields []string) KVPairs {
	newpairs := make(KVPairs)
	for _, field := range fields {
		newpairs[field] = pairs[field]
	}
	return newpairs
}

func Select(pairsList []KVPairs, fields []string) []KVPairs {
	result := make([]KVPairs, len(pairsList))
	for i, pairs := range pairsList {
		result[i] = SelectSingle(pairs, fields)
	}
	return result
}

func SelectWithoutSingle(pairs KVPairs, fieldToOmit string) KVPairs {
	newpairs := make(KVPairs)
	for field := range pairs {
		if field != fieldToOmit {
			newpairs[field] = pairs[field]
		}
	}
	return newpairs
}

func SelectWithout(pairsList []KVPairs, fieldToOmit string) []KVPairs {
	result := make([]KVPairs, len(pairsList))
	for i, pairs := range pairsList {
		result[i] = SelectWithoutSingle(pairs, fieldToOmit)
	}
	return result
}

func SplitIntoSharedAndDistinct(pairsList []KVPairs) ([]string, KVPairs, []string, []KVPairs) {
	sharedFields, distinctFields := IdentifySharedFields(pairsList)
	shared := SelectSingle(pairsList[0], sharedFields)
	projection := Select(pairsList, distinctFields)
	return sharedFields, shared, distinctFields, projection
}

func IdentifySharedFields(pairsList []KVPairs) (sharedFields []string, distinctFields []string) {
	// populate map with the set of unique values
	valuesPerField := make(map[string]StrSet)
	for _, record := range pairsList {
		for field, value := range record {
			s, ok := valuesPerField[field]
			if !ok {
				s = NewStrSet()
				valuesPerField[field] = s
			}
			s.Add(value)
		}
	}

	sharedFieldsSet := NewStrSet()
	for field, values := range valuesPerField {
		if values.Len() == 1 {
			sharedFieldsSet.Add(field)
		}
	}

	for field := range valuesPerField {
		if sharedFieldsSet.In(field) {
			sharedFields = append(sharedFields, field)
		} else {
			distinctFields = append(distinctFields, field)
		}
	}
	return sharedFields, distinctFields
}
