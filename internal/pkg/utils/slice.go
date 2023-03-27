package utils

func StringContains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}
