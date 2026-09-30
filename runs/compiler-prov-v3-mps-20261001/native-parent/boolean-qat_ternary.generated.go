package tiny_smoke

//gooo:generated:start id="tiny-smoke://activity/boolean-hole" kind="activity"
func BooleanHole(input int64) int64 {
	var result int64 = 0
	if input <= 0 {
		result = result + 1
	} else {
		result = result - 1
	}
	result = result + 1
	return result
}

//gooo:generated:end id="tiny-smoke://activity/boolean-hole" kind="activity"
