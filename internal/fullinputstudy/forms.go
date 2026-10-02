// Package fullinputstudy defines the frozen complete-input interventions.
package fullinputstudy

import (
	"errors"
	"strconv"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

const Protocol = "docs/full-input-judgment-preregistration-20261003.md"

var TrainingForms = [5]string{"original", "request-prefix", "please-prefix", "request-suffix", "please-suffix"}
var EvaluationForms = [2]string{"task-prefix", "complete-suffix"}

type Input struct {
	Text      string `json:"text"`
	PartBytes [3]int `json:"part_bytes"`
	Declined  bool   `json:"representation_declined"`
}

func Apply(text, language, form string) (Input, error) {
	if language != "en" && language != "ko" {
		return Input{}, errors.New("known bilingual language required")
	}
	var original [jointdecision.ThreeFeatureDim]float32
	if err := jointdecision.FeaturesIntoThree(text, &original); err != nil {
		return Input{}, err
	}
	prefix, suffix, err := addition(language, form)
	if err != nil {
		return Input{}, err
	}
	parts, err := jointdecision.ThreeParts(text)
	if err != nil {
		return Input{}, err
	}
	var result Input
	var joined strings.Builder
	joined.WriteString("gooo;joint3|")
	for i, part := range parts {
		header, intent, found := strings.Cut(part, ";intent: ")
		if !found {
			return Input{}, errors.New("semantic header missing")
		}
		full := header + ";intent: " + prefix + intent + suffix
		result.PartBytes[i] = len(full)
		joined.WriteString(strconv.Itoa(len(full)))
		joined.WriteByte(':')
		joined.WriteString(full)
	}
	result.Text = joined.String()
	var candidate [jointdecision.ThreeFeatureDim]float32
	if err = jointdecision.FeaturesIntoThree(result.Text, &candidate); err != nil {
		result.Declined = true
		return result, nil
	}
	for i, value := range original {
		if i%256 < 64 && candidate[i] != value {
			return Input{}, errors.New("wording changed source coordinates")
		}
	}
	if err = jointdecision.FeaturesIntoThreeBag(result.Text, &candidate); err != nil {
		return Input{}, err
	}
	for i, value := range original {
		if i%256 < 64 && candidate[i] != value {
			return Input{}, errors.New("v4 changed source coordinates")
		}
	}
	return result, nil
}

func addition(language, form string) (string, string, error) {
	ko := language == "ko"
	switch form {
	case "original":
		return "", "", nil
	case "request-prefix":
		if ko {
			return "요청: ", "", nil
		}
		return "Request: ", "", nil
	case "please-prefix":
		if ko {
			return "다음 지시를 따라 주세요: ", "", nil
		}
		return "Please follow this instruction: ", "", nil
	case "request-suffix":
		if ko {
			return "", " 이것이 요청입니다.", nil
		}
		return "", " This is the request.", nil
	case "please-suffix":
		if ko {
			return "", " 이 지시를 따라 주세요.", nil
		}
		return "", " Please follow this instruction.", nil
	case "task-prefix":
		if ko {
			return "이 작업에서는, ", "", nil
		}
		return "For this task, ", "", nil
	case "complete-suffix":
		if ko {
			return "", " 이것이 전체 지시입니다.", nil
		}
		return "", " That is the complete instruction.", nil
	default:
		return "", "", errors.New("unknown frozen wording form")
	}
}
