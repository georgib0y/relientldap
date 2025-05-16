package util

func CloneMap[K comparable, V any](m map[K]V) map[K]V {
	cloned := map[K]V{}

	for k, v := range m {
		cloned[k] = v
	}

	return cloned
}

func CloneMapNested[K1, K2 comparable, V any](m map[K1]map[K2]V) map[K1]map[K2]V {
	cloned := map[K1]map[K2]V{}

	for k, v := range m {
		cloned[k] = CloneMap(v)
	}

	return cloned
}

func CmpMapKeys[K comparable, V any](m1, m2 map[K]V) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k := range m1 {
		if _, ok := m2[k]; !ok {
			return false
		}
	}

	return true
}
