package arrays

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
func AppendUnique(slice []string, items ...string) []string {
	seen := make(map[string]bool)
	for _, s := range slice {
		seen[s] = true
	}
	for _, i := range items {
		if !seen[i] {
			slice = append(slice, i)
			seen[i] = true
		}
	}
	return slice
}

func Remove(slice []string, target string) []string {
	result := []string{}
	for _, item := range slice {
		if item != target {
			result = append(result, item)
		}
	}
	return result
}
