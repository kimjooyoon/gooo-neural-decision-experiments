package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

var forms = [...]string{"original", "bare", "calibration_prefix", "development_prefix", "development_suffix"}

func wrapper(split, language string) (string, error) {
	if language != "en" && language != "ko" {
		return "", fmt.Errorf("unknown language %q", language)
	}
	switch split {
	case "train":
		return "", nil
	case "calibration":
		if language == "en" {
			return "Decision request: ", nil
		}
		return "구성 요청: ", nil
	case "development":
		if language == "en" {
			return "While composing, ", nil
		}
		return "함수를 구성할 때, ", nil
	default:
		return "", fmt.Errorf("unknown split %q", split)
	}
}

func transformed(v threecohort.View, form string) (string, error) {
	if form == "original" {
		return v.Text, nil
	}
	old, err := wrapper(v.Split, v.Language)
	if err != nil {
		return "", err
	}
	cal, _ := wrapper("calibration", v.Language)
	dev, _ := wrapper("development", v.Language)
	parts := v.Parts
	for i, part := range parts {
		header, natural, found := strings.Cut(part, ";intent: ")
		if !found || !strings.HasPrefix(natural, old) {
			return "", fmt.Errorf("exact authored wrapper missing")
		}
		body := strings.TrimPrefix(natural, old)
		if body == "" {
			return "", fmt.Errorf("empty authored instruction")
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
		default:
			return "", fmt.Errorf("unknown input form %q", form)
		}
		parts[i] = header + ";intent: " + natural
	}
	return jointdecision.EncodeThree(parts)
}

func featurePin(text string, original *[768]float32) (string, float64, error) {
	var features [768]float32
	if err := jointdecision.FeaturesIntoThree(text, &features); err != nil {
		return "", 0, err
	}
	var raw [768 * 4]byte
	var distance float64
	for i, f := range features {
		if i%256 < 64 && f != original[i] {
			return "", 0, fmt.Errorf("source channel changed")
		}
		binary.LittleEndian.PutUint32(raw[i*4:], math.Float32bits(f))
		d := float64(f) - float64(original[i])
		distance += d * d
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw[:])), math.Sqrt(distance), nil
}

func passingSet(v threecohort.View) uint8 {
	var result uint8
	for mask, n := range v.Target.Passed {
		if n == v.Target.Cases {
			result |= 1 << mask
		}
	}
	return result
}
