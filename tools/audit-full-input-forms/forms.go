package main

import (
	"encoding/binary"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/fullinputstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

var forms = [11]string{"original", "request-prefix", "please-prefix", "request-suffix", "please-suffix", "task-prefix", "complete-suffix", "bare", "calibration_prefix", "development_prefix", "development_suffix"}

func authoredWrapper(split, language string) (string, error) {
	if language != "en" && language != "ko" {
		return "", errors.New("unknown language")
	}
	switch split {
	case "train":
		return "", nil
	case "calibration":
		if language == "ko" {
			return "구성 요청: ", nil
		}
		return "Decision request: ", nil
	case "development":
		if language == "ko" {
			return "함수를 구성할 때, ", nil
		}
		return "While composing, ", nil
	default:
		return "", errors.New("unknown split")
	}
}

// Only explicitly named diagnostic controls replace the frozen authored prefix.
// All additions preserve the entire original instruction, including feedback.
func applyForm(v threecohort.View, form string) (fullinputstudy.Input, error) {
	for _, allowed := range forms[:7] {
		if form == allowed {
			return fullinputstudy.Apply(v.Text, v.Language, form)
		}
	}
	allowed := false
	for _, name := range forms[7:] {
		allowed = allowed || form == name
	}
	if !allowed {
		return fullinputstudy.Input{}, errors.New("unknown audit form")
	}
	old, err := authoredWrapper(v.Split, v.Language)
	if err != nil {
		return fullinputstudy.Input{}, err
	}
	cal, _ := authoredWrapper("calibration", v.Language)
	dev, _ := authoredWrapper("development", v.Language)
	parts, err := jointdecision.ThreeParts(v.Text)
	if err != nil {
		return fullinputstudy.Input{}, err
	}
	result := fullinputstudy.Input{}
	var joined strings.Builder
	joined.WriteString("gooo;joint3|")
	for i, part := range parts {
		header, natural, found := strings.Cut(part, ";intent: ")
		if !found || !strings.HasPrefix(natural, old) {
			return result, errors.New("original authored prefix differs")
		}
		body := strings.TrimPrefix(natural, old)
		if body == "" {
			return result, errors.New("empty authored instruction")
		}
		switch form {
		case "bare":
			natural = body
		case "calibration_prefix":
			natural = cal + body
		case "development_prefix":
			natural = dev + body
		case "development_suffix":
			natural = body + " " + strings.TrimSuffix(dev, ", ") + "."
		}
		full := header + ";intent: " + natural
		result.PartBytes[i] = len(full)
		joined.WriteString(strconv.Itoa(len(full)))
		joined.WriteByte(':')
		joined.WriteString(full)
	}
	result.Text = joined.String()
	var feature [jointdecision.ThreeFeatureDim]float32
	result.Declined = jointdecision.FeaturesIntoThree(result.Text, &feature) != nil
	return result, nil
}

func features(text string, bag bool) ([jointdecision.ThreeFeatureDim]float32, error) {
	var out [jointdecision.ThreeFeatureDim]float32
	if bag {
		err := jointdecision.FeaturesIntoThreeBag(text, &out)
		return out, err
	}
	err := jointdecision.FeaturesIntoThree(text, &out)
	return out, err
}

func preservedSource(original, candidate *[jointdecision.ThreeFeatureDim]float32) bool {
	for i, v := range original {
		if i%256 < 64 && math.Float32bits(v) != math.Float32bits(candidate[i]) {
			return false
		}
	}
	return true
}

func featureBytes(values *[jointdecision.ThreeFeatureDim]float32) [jointdecision.ThreeFeatureDim * 4]byte {
	var raw [jointdecision.ThreeFeatureDim * 4]byte
	for i, v := range values {
		binary.LittleEndian.PutUint32(raw[i*4:], math.Float32bits(v))
	}
	return raw
}
