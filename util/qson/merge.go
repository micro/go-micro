package qson

// merge merges a with b if they are either both slices
// or map[string]interface{} types. Otherwise it returns b.
func merge(a interface{}, b interface{}) interface{} {
	switch aT := a.(type) {
	case map[string]interface{}:
		return mergeMap(aT, b.(map[string]interface{}))
	case []interface{}:
		return mergeSlice(aT, b.([]interface{}))
	default:
		return b
	}
}

// mergeMap merges a with b, attempting to merge any nested
// values in nested maps but eventually overwriting anything
// in a that can't be merged with whatever is in b.
func mergeMap(a map[string]interface{}, b map[string]interface{}) map[string]interface{} {
	for bK, bV := range b {
		if _, ok := a[bK]; ok {
			a[bK] = merge(a[bK], bV)
		} else {
			a[bK] = bV
		}
	}
	return a
}

// mergeSlice merges a with b and returns the result.
func mergeSlice(a []interface{}, b []interface{}) []interface{} {
	a = append(a, b...)
	return a
}
