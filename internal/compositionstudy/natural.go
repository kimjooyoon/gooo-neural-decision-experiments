package compositionstudy

import "fmt"

func Split(config int) string {
	if config < 32 {
		return "train"
	}
	if config < 40 {
		return "calibration"
	}
	return "development"
}

// TemplateID is disjoint by split before collection. Language and both
// feature arms of the same contract/config remain in the same split.
func TemplateID(kind string, config int, language string) string {
	return fmt.Sprintf("fresh-v1-%s-%s-%s", Split(config), kind, language)
}

func Natural(family, kind string, coordinate, config, desired int, language string) string {
	bit := (desired >> coordinate) & 1
	sourceBit := (config % 4 >> coordinate) & 1
	reverse := bit != sourceBit
	var text string
	switch kind {
	case "local_reference":
		if language == "en" {
			text = [2]string{"Read the first declared computed local.", "Read the second declared computed local."}[bit]
		} else {
			text = [2]string{"먼저 선언한 계산 변수를 읽어라.", "나중에 선언한 계산 변수를 읽어라."}[bit]
		}
	case "assignment_target":
		if language == "en" {
			text = [2]string{"Write into the first declared local.", "Write into the second declared local."}[bit]
		} else {
			text = [2]string{"먼저 선언한 변수에 대입하라.", "나중에 선언한 변수에 대입하라."}[bit]
		}
	case "operand_order":
		operation, ko := "subtraction", "뺄셈"
		if family == "predicate_branch" {
			operation, ko = "comparison", "비교"
		}
		if language == "en" {
			text = "Keep the " + operation + " operands as in the source."
			if reverse {
				text = "Swap the " + operation + " operands of the source."
			}
		} else {
			text = "원문의 " + ko + " 피연산자 순서를 유지하라."
			if reverse {
				text = "원문의 " + ko + " 피연산자 순서를 바꾸어라."
			}
		}
	case "branch_layout":
		if language == "en" {
			text = "Keep the conditional arms in the source."
			if reverse {
				text = "Swap the conditional arms of the source."
			}
		} else {
			text = "원문의 조건 분기 바디를 유지하라."
			if reverse {
				text = "원문의 조건 분기 바디를 교환하라."
			}
		}
	case "root_order":
		if language == "en" {
			text = "Keep the two writes in source order."
			if reverse {
				text = "Exchange the two writes in the source order."
			}
		} else {
			text = "원문의 두 쓰기 순서를 유지하라."
			if reverse {
				text = "원문의 두 쓰기 실행 순서를 교환하라."
			}
		}
	}
	// Templates are fixed here before any collection/training or observations.
	if Split(config) == "calibration" {
		if language == "en" {
			text = "Requested action: " + text
		} else {
			text = "요청한 동작: " + text
		}
	} else if Split(config) == "development" {
		if language == "en" {
			text = "Build the function so that you " + text
		} else {
			text = "함수를 조립할 때 " + text
		}
	}
	return text
}
