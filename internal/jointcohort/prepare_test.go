package jointcohort

import "testing"

func TestMissingSourcePairDeclines(t *testing.T) {
	if _, err := prepare([2]*sourceRow{}); err == nil {
		t.Fatal("missing source accepted")
	}
}

func TestDifferentSourceCoordinatesDeclineBeforeFixture(t *testing.T) {
	a, b := &sourceRow{Group: "one", Language: "en"}, &sourceRow{Group: "two", Language: "en"}
	if _, err := prepare([2]*sourceRow{a, b}); err == nil {
		t.Fatal("mixed source accepted")
	}
}
