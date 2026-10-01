package main

import (
	"encoding/json"
	"errors"
	"sort"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func ownContextNames() ([]string, error) {
	root := "runs/own-model-sdk-context-20261001/"
	pins := map[string]string{
		root + "preexecution.json": "c5af58371ea798b5499c0d9d1490d931b4c38d3c31006a2e00cae8ff6b744ed3",
		root + "report.json":       "153b5c1475c6c9a15e5325bb3cb0588d19ae134cb6c43aeedf1f5064cab9ffd8",
		root + "audit.json":        "6cfa8d309a92f206f97d1e4c1426e48a8024f1b6c43e2c3ae24830a2e7c34091",
		"publication/own-model-context-sdk-release-20261001.json": "92183fca74da8b686829a5c44225d83928935968b67ef3fda1607fa92a6d60f2",
	}
	names := []string{"docs/own-model-sdk-context-preregistration.md"}
	for name, pin := range pins {
		raw, err := read(name)
		if err != nil || hash(raw) != pin {
			return nil, errors.New("own-model context frozen evidence differs")
		}
		names = append(names, name)
	}
	models := map[string]string{
		"fp32":        "7998ca6e5cbe28455e0467a19bc77c03f99683d8623f35ebe95f79e905624fbd",
		"ptq_ternary": "2f996c11983485c4e2fb1e9c2b08444042d8725a3bf0947025b318014d28ea50",
		"qat_ternary": "6a0162e814c589ac833b3f3ac2b7eeff4dd48ad75a015ee822523c534995196d",
	}
	calls, cases, evaluated, extra, wrong := 0, 0, 0, 0, 0
	for _, language := range []string{"en", "ko"} {
		for _, contract := range []string{"sparse", "full"} {
			fixture := "studies/own-model-sdk-context-v1/" + language + "-" + contract + ".json"
			raw, err := read(fixture)
			var document struct {
				Plan pathplan.Plan `json:"path_plan"`
			}
			if err != nil || json.Unmarshal(raw, &document) != nil {
				return nil, errors.New("own-model context typed document unavailable")
			}
			prepared, err := pathplan.Prepare(document.Plan)
			if err != nil {
				return nil, err
			}
			names = append(names, fixture)
			for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary", "offline"} {
				name := root + language + "-" + contract + "-" + variant + ".json"
				capture, err := read(name)
				var value struct {
					Schema  string                `json:"schema"`
					Input   string                `json:"input_document_sha256"`
					Plan    string                `json:"original_plan_sha256"`
					Body    string                `json:"gooo_body"`
					Applied bool                  `json:"context_applied"`
					Native  int                   `json:"native_compiler_calls"`
					Go      int                   `json:"emitted_go_processes"`
					Search  pathplan.SearchResult `json:"search"`
				}
				if err != nil || json.Unmarshal(capture, &value) != nil || value.Schema != "gooo/own-model-typed-example/v1" || value.Input != hash(raw) || value.Plan != prepared.PlanSHA256() || value.Native != 0 || value.Go != 0 {
					return nil, errors.New("own-model SDK capture binding differs")
				}
				s := value.Search
				connected := variant != "offline"
				if value.Applied != connected || s.Selection.ModelCalls != boolCount(connected) || s.Selection.ExternalCalls != 0 || !s.Selection.ExternalCallsKnown || s.Selection.MetadataSHA256 != models[variant] {
					return nil, errors.New("own-model SDK prediction accounting differs")
				}
				if len(s.Selection.Choices) != 1 || s.SelectedTrainingPassed != s.TrainingTotal || s.TrainingTotal != 1+boolCount(contract == "full") || s.Evaluated != len(s.Attempts) || value.Body != ownContextBody(s.Selection.Choices["operands"]) {
					return nil, errors.New("own-model SDK selected finite body differs")
				}
				calls += s.Selection.ModelCalls
				cases += s.TrainingTotal
				if contract == "full" {
					if value.Body != "return (input - 2)" {
						return nil, errors.New("full intention not attained")
					}
					extra += s.Evaluated - 1
				} else if value.Body != "return (input - 2)" {
					wrong++
				}
				for _, attempt := range s.Attempts {
					label := attempt.Choices["operands"]
					if ownContextBody(label) == "" || len(attempt.Results) != s.TrainingTotal {
						return nil, errors.New("candidate path/results differ")
					}
					passed := 0
					for i, result := range attempt.Results {
						actual := result.Input - 2
						if label == "layout_reverse" {
							actual = 2 - result.Input
						}
						if result.Input != int64(2+i) || result.Expected != result.Input-2 || result.Actual != actual || result.Passed != (actual == result.Expected) {
							return nil, errors.New("independent candidate arithmetic differs")
						}
						passed += boolCount(result.Passed)
						evaluated++
					}
					if attempt.Passed != passed || attempt.Total != s.TrainingTotal {
						return nil, errors.New("candidate case totals differ")
					}
				}
				names = append(names, name)
			}
		}
	}
	if len(names) != 25 || calls != 12 || cases != 24 || evaluated != 30 || extra != 3 || wrong != 3 {
		return nil, errors.New("own-model context frozen denominators differ")
	}
	sort.Strings(names)
	return names, nil
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}
func ownContextBody(label string) string {
	if label == "layout_forward" {
		return "return (input - 2)"
	}
	if label == "layout_reverse" {
		return "return (2 - input)"
	}
	return ""
}
