package tiny_smoke

//gooo:generated:start id="tiny-smoke://activity/arithmetic-hole" kind="activity"
func ArithmeticHole(input int64) int64 {
	var result = (input + 2)
	if input < 0 {
		result = result + 1
	} else {
		result = result - 1
	}
	result = result + 1
	return result
}

//gooo:generated:end id="tiny-smoke://activity/arithmetic-hole" kind="activity"
