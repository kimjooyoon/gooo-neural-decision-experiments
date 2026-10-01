package main

import "path/filepath"

func ownContextInventory(output string, rows []ownContextRow) (map[string]string, error) {
	names := []string{filepath.Join(output, "preexecution.json"), ownNativeContextProtocol}
	for _, name := range []string{"subtraction.gooo.fixture", "conditional.gooo.fixture", "compound.gooo.fixture",
		"conditional-en.json", "conditional-ko.json", "compound.json"} {
		names = append(names, ownNativeContextRoot+name)
	}
	for _, id := range []string{"en-sparse", "en-full", "ko-sparse", "ko-full"} {
		names = append(names, "studies/own-model-sdk-context-v1/"+id+".json")
	}
	for _, row := range rows {
		for _, arm := range []string{"fp32", "ptq_ternary", "qat_ternary", "offline"} {
			id := row.ID + "-" + arm
			for _, suffix := range []string{".json", "-metrics.json", "-execution.json"} {
				names = append(names, filepath.Join(output, id+suffix))
			}
		}
	}
	files := make(map[string]string, len(names))
	for _, name := range names {
		raw, err := read(name)
		if err != nil {
			return nil, err
		}
		files[name] = hash(raw)
	}
	return files, nil
}
