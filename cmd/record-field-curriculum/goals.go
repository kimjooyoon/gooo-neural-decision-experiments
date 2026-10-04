package main

func titleCopiesState(family int) bool {
	return family == 2 || family == 4 || family == 6 || family == 7
}

func goalValues(family, goal int, title, state, reason string) map[string]string {
	fields := names[family]
	want := [3]string{title, "ready", reason + ":accepted"}
	if goal&1 != 0 {
		want[0] = "draft"
		if titleCopiesState(family) {
			want[0] = state
		}
	}
	if goal&2 != 0 {
		want[1] = "wait"
	}
	if goal&4 != 0 {
		want[2] = "deferred"
		if family >= 3 {
			want[2] = ":accepted" + reason
		}
	}
	return map[string]string{fields[0]: want[0], fields[1]: want[1], fields[2]: want[2]}
}

func goalMask(order [3]int, orientation uint16, goal int) uint16 {
	target := orientation
	for position, role := range order {
		if goal&(1<<role) != 0 {
			target ^= 1 << position
		}
	}
	return target
}

func goalIntent(family, role int, opposite bool, language string, style int) string {
	lang := 0
	if language == "en" {
		lang = 1
	}
	if !opposite && style == 0 {
		return intents[lang][role]
	}
	// Eight semantic requirements. Alternate requirements depend on actual source
	// expressions rather than assuming every family copies state or prefixes text.
	kind := [3]int{0, 3, 5}[role]
	if opposite {
		kind = [3]int{1, 4, 6}[role]
		if role == 0 && titleCopiesState(family) {
			kind = 2
		}
		if role == 2 && family >= 3 {
			kind = 7
		}
	}
	words := [2][3][8]string{
		{
			{"원래 제목을 그대로 사용한다.", "제목을 draft 문자열로 설정한다.", "제목에 입력 상태 필드의 값을 사용한다.", "상태를 ready 문자열로 설정한다.", "상태를 wait 문자열로 설정한다.", "기존 사유 뒤에 :accepted를 붙인다.", "사유를 deferred 문자열로 설정한다.", "기존 사유 앞에 :accepted를 붙인다."},
			{"제목은 입력값을 유지한다.", "draft를 제목 값으로 넣는다.", "입력에 담긴 상태를 제목으로 복사한다.", "ready를 상태 값으로 넣는다.", "wait를 상태 값으로 넣는다.", "사유의 끝에 :accepted를 덧붙인다.", "deferred를 사유 값으로 넣는다.", "사유의 시작에 :accepted를 덧붙인다."},
			{"입력 제목을 변경하지 않고 전달한다.", "제목 필드에 상수 draft를 사용한다.", "제목 필드를 입력의 상태 값으로 채운다.", "상태 필드에 상수 ready를 사용한다.", "상태 필드에 상수 wait를 사용한다.", "사유를 먼저 놓고 :accepted를 뒤에 연결한다.", "사유 필드에 상수 deferred를 사용한다.", ":accepted를 먼저 놓고 사유를 뒤에 연결한다."},
		},
		{
			{"Keep the original title.", "Set title to the draft string.", "Use the input state field as the title.", "Set state to the ready string.", "Set state to the wait string.", "Append :accepted to the original reason.", "Set reason to the deferred string.", "Prefix :accepted to the original reason."},
			{"Preserve the input title.", "Put draft in the title field.", "Copy the input state into the title.", "Put ready in the state field.", "Put wait in the state field.", "Add :accepted at the end of the reason.", "Put deferred in the reason field.", "Add :accepted at the start of the reason."},
			{"Return the title without changing its value.", "The title should contain the literal draft.", "Fill the title with the state from the input.", "The state should contain the literal ready.", "The state should contain the literal wait.", "Place the reason first and :accepted after it.", "The reason should contain the literal deferred.", "Place :accepted first and the reason after it."},
		},
	}
	return words[lang][style][kind]
}
