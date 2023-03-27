package utils

func CoalesceZero(values ...int64) int64 {
	for _, v := range values {
		if v != Zero {
			return v
		}
	}
	return Zero
}
