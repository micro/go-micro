package flow

func include(slice []string, f string) bool {
	for _, s := range slice {
		if s == f {
			return true
		}
	}
	return false
}
