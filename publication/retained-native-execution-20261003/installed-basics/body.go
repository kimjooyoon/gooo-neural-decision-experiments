package bodycodegen

//gooo:generated:start id="bodycodegen://activity/clamp-below-zero" kind="activity"
func ClampBelowZero(input int64) int64 {
	var value = input
	value = input
	var accepted = value >= 0
	if accepted {
		return value
	} else {
		return 0
	}
}

//gooo:generated:end id="bodycodegen://activity/clamp-below-zero" kind="activity"
