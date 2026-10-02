package threecompositionstudy

import (
	"fmt"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func Split(config int) string {
	if config < 16 {
		return "train"
	}
	if config < 20 {
		return "calibration"
	}
	return "development"
}

func TemplateID(kind string, config int, language string) string {
	return fmt.Sprintf("three-v1-%s-%s-%s", Split(config), kind, language)
}

// Natural describes the requested source edit, not an option label or goal mask.
// Templates and split boundaries are fixed before model observation.
func Natural(family, kind string, coordinate, config, goal int, language string) string {
	bit := (goal >> coordinate) & 1
	reverse := bit != (FallbackMask(config)>>coordinate)&1
	var text string
	switch kind {
	case pathplan.LocalReference:
		if family == "boolean_reference_branch_operand" {
			if language == "en" {
				text = [2]string{"Read the first declared Boolean local.", "Read the second declared Boolean local."}[bit]
			} else {
				text = [2]string{"먼저 선언한 불리언 변수를 읽어라.", "나중에 선언한 불리언 변수를 읽어라."}[bit]
			}
		} else if language == "en" {
			text = [2]string{"Read the first declared integer local.", "Read the second declared integer local."}[bit]
		} else {
			text = [2]string{"먼저 선언한 정수 변수를 읽어라.", "나중에 선언한 정수 변수를 읽어라."}[bit]
		}
	case pathplan.AssignmentTarget:
		if language == "en" {
			text = [2]string{"Write into the first declared integer local.", "Write into the second declared integer local."}[bit]
		} else {
			text = [2]string{"먼저 선언한 정수 변수에 대입하라.", "나중에 선언한 정수 변수에 대입하라."}[bit]
		}
	case pathplan.OperandOrder:
		operation, ko := "subtraction", "뺄셈"
		if family == "comparison_branch_assignment" {
			operation, ko = "comparison", "비교"
		}
		if language == "en" {
			text = "Keep the source " + operation + " operands."
			if reverse {
				text = "Swap the source " + operation + " operands."
			}
		} else {
			text = "원문의 " + ko + " 피연산자 순서를 유지하라."
			if reverse {
				text = "원문의 " + ko + " 피연산자 순서를 바꾸어라."
			}
		}
	case pathplan.BranchLayout:
		if language == "en" {
			text = "Keep the source conditional arms."
			if reverse {
				text = "Swap the source conditional arms."
			}
		} else {
			text = "원문의 조건 분기 바디를 유지하라."
			if reverse {
				text = "원문의 조건 분기 바디를 교환하라."
			}
		}
	case pathplan.RootOrder:
		if language == "en" {
			text = "Keep the two source writes in order."
			if reverse {
				text = "Exchange the order of the two source writes."
			}
		} else {
			text = "원문의 두 쓰기 실행 순서를 유지하라."
			if reverse {
				text = "원문의 두 쓰기 실행 순서를 교환하라."
			}
		}
	}
	if Split(config) == "calibration" {
		if language == "en" {
			text = "Decision request: " + text
		} else {
			text = "구성 요청: " + text
		}
	} else if Split(config) == "development" {
		if language == "en" {
			text = "While composing, " + text
		} else {
			text = "함수를 구성할 때, " + text
		}
	}
	return text
}
