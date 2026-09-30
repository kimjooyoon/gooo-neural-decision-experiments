package main

import "testing"

const firstRow = `{"id":"a","template_id":"group","configuration_id":"plain","language":"en","split":"train","text":"add two numbers","label":"add"}` + "\n"
const secondRow = `{"id":"b","template_id":"other","configuration_id":"plain","language":"en","split":"test","text":"compare two numbers","label":"less_than"}` + "\n"

func TestDatasetExplicitCountRetainsV1Default(t *testing.T) {
	raw := []byte(firstRow + secondRow)
	if _, err := decodeDataset(raw); err == nil {
		t.Fatal("v1 default accepted two rows")
	}
	if rows, err := decodeDatasetWithRows(raw, 2); err != nil || len(rows) != 2 {
		t.Fatalf("explicit count: %v", err)
	}
	for _, count := range []int{0, 1, 3, 8193} {
		if _, err := decodeDatasetWithRows(raw, count); err == nil {
			t.Fatalf("accepted count %d", count)
		}
	}
}

func TestDatasetRejectsDuplicateAndTemplateLeakage(t *testing.T) {
	if _, err := decodeDatasetWithRows([]byte(firstRow+firstRow), 2); err == nil {
		t.Fatal("accepted duplicate")
	}
	leaked := `{"id":"b","template_id":"group","configuration_id":"other","language":"en","split":"test","text":"another instruction","label":"add"}` + "\n"
	if _, err := decodeDatasetWithRows([]byte(firstRow+leaked), 2); err == nil {
		t.Fatal("accepted template leakage")
	}
}
