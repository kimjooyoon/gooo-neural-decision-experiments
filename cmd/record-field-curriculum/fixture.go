package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

var orders = [6][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
var names = [8][3]string{{"title", "state", "reason"}, {"caption", "status", "explanation"}, {"heading", "stage", "note"}, {"label", "phase", "cause"}, {"subject", "mode", "why"}, {"summary", "condition", "detail"}, {"message", "flag", "context"}, {"display", "status", "comment"}}
var intents = [2][3]string{{"원래 제목을 그대로 사용한다.", "상태를 ready 문자열로 설정한다.", "기존 사유 뒤에 :accepted를 붙인다."}, {"Keep the original title.", "Set state to the ready string.", "Append :accepted to the original reason."}}

func fixture(family int, order [3]int, mask uint16, language string) []byte {
	fields := names[family]
	good := [3]string{"input0." + fields[0], strconv.Quote("ready"), "input0." + fields[2] + " + " + strconv.Quote(":accepted")}
	bad := [3]string{strconv.Quote("draft"), strconv.Quote("wait"), strconv.Quote("deferred")}
	if family == 2 || family == 6 {
		good[0] += ` + ""`
	}
	if family == 3 || family == 7 {
		good[0] = `"" + ` + good[0]
	}
	if family == 2 || family == 4 || family == 6 || family == 7 {
		bad[0] = "input0." + fields[1]
	}
	if family == 5 {
		good[0] = "(" + good[0] + ")"
	}
	if family >= 3 {
		bad[2] = strconv.Quote(":accepted") + " + input0." + fields[2]
	}
	first, second := good, bad
	for position, role := range order {
		if mask&(1<<position) != 0 {
			first[role], second[role] = bad[role], good[role]
		}
	}
	constructor := fmt.Sprintf("Candidate{%s: %s, %s: %s, %s: %s}", fields[0], first[0], fields[1], first[1], fields[2], first[2])
	body := []string{
		fmt.Sprintf("let copy = input0\nif input1 && copy.%s != \"ready\" {\n    copy = %s\n} else {\n    copy = input0\n}\nreturn copy", fields[1], constructor),
		fmt.Sprintf("if !input1 || input0.%s == \"ready\" {\n    return input0\n}\nreturn %s", fields[1], constructor),
		fmt.Sprintf("let copy = input0\nif input1 {\n    if copy.%s != \"ready\" {\n        copy = %s\n    }\n}\nreturn copy", fields[1], constructor),
		fmt.Sprintf("if input1 && input0.%s != \"ready\" {\n    return %s\n}\nreturn input0", fields[1], constructor),
		fmt.Sprintf("let chosen = input0\nif !(input1 && chosen.%s != \"ready\") {\n    return chosen\n}\nchosen = %s\nreturn chosen", fields[1], constructor),
		fmt.Sprintf("let result = input0\nif input1 == true && input0.%s != \"ready\" {\n    result = %s\n}\nreturn result", fields[1], constructor),
		fmt.Sprintf("if input0.%s == \"ready\" {\n    return input0\n}\nif input1 {\n    return %s\n}\nreturn input0", fields[1], constructor),
		fmt.Sprintf("let ready = input0.%s == \"ready\"\nlet active = input1 && !ready\nif active {\n    return %s\n}\nreturn input0", fields[1], constructor),
	}[family]
	var b strings.Builder
	fmt.Fprintf(&b, "package records\nnamespace fieldstudy\nentity Text id \"fieldstudy://text\"\nentity Boolean id \"fieldstudy://boolean\"\nentity Candidate id \"fieldstudy://family/%d/record\" fields {\n", family)
	for role, field := range fields {
		fmt.Fprintf(&b, "    field %s id \"fieldstudy://family/%d/field/%d\" type string required one\n", field, family, role)
	}
	fmt.Fprintf(&b, "}\nactivity Select(Candidate, Boolean) -> Candidate computes `%s` assembling {\n", body)
	lang := 0
	if language == "en" {
		lang = 1
	}
	for _, role := range order {
		fmt.Fprintf(&b, "    choice %s field_value at %s alternative %s intent %s\n", strconv.Quote(fmt.Sprintf("role-%d", role)), strconv.Quote(strconv.Itoa(role)), strconv.Quote(second[role]), strconv.Quote(intents[lang][role]))
	}
	for _, c := range [5]struct {
		title, state, reason string
		active               bool
	}{{"한글", "queued", "검토", true}, {"English", "queued", "review", true}, {"", "queued", "", true}, {"kept", "queued", "later", false}, {"already", "ready", "done", true}} {
		input := map[string]string{fields[0]: c.title, fields[1]: c.state, fields[2]: c.reason}
		expected := map[string]string{fields[0]: c.title, fields[1]: c.state, fields[2]: c.reason}
		if c.active && c.state != "ready" {
			expected[fields[1]] = "ready"
			expected[fields[2]] = c.reason + ":accepted"
		}
		in, _ := json.Marshal([]any{input, c.active})
		out, _ := json.Marshal(expected)
		fmt.Fprintf(&b, "    value_case %s -> %s\n", strconv.Quote(string(in)), strconv.Quote(string(out)))
	}
	fmt.Fprintf(&b, "    attempts \"8\"\n}\nactivity Label(Candidate) -> Text computes `return input.%s + \":\" + input.%s + \":\" + input.%s`\nbind Select.result -> Label.input\n", fields[0], fields[1], fields[2])
	return []byte(b.String())
}
