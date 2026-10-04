package main

import (
	"fmt"
	"strconv"
	"strings"
)

func shapes() []shape {
	return []shape{
		{"ordered", "let copy = input0\ncopy.title = \"draft\"\ncopy.state = \"wait\"\ncopy.reason = \"deferred\"\nreturn copy", `copy.title + ":" + copy.state`, [3]int{0, 1, 2}, false, func(r record) string { return r.Title + ":ready" }},
		{"record_snapshot", "let copy = input0\nlet saved = copy\ncopy.title = \"draft\"\ncopy.state = \"wait\"\ncopy.reason = saved.state\nreturn copy", `copy.title + ":" + saved.state`, [3]int{0, 1, 2}, false, func(r record) string { return r.Title + ":" + r.State }},
		{"guarded", "let copy = input0\nif input1 && copy.state != \"ready\" {\ncopy.title = \"draft\"\ncopy.state = \"wait\"\ncopy.reason = \"deferred\"\n}\nreturn copy", `copy.title + ":" + copy.state`, [3]int{0, 1, 2}, true, func(r record) string { return r.Title + ":ready" }},
		{"scalar_snapshot", "let copy = input0\ncopy.title = \"draft\"\nlet saved = copy.title\ncopy.state = \"wait\"\ncopy.reason = saved\nreturn copy", `saved + ":" + copy.state`, [3]int{0, 1, 2}, false, func(r record) string { return r.Title + ":ready" }},
		{"self_append", "let copy = input0\ncopy.title = \"draft\"\ncopy.state = \"wait\"\ncopy.reason = copy.reason\nreturn copy", `copy.reason + ":accepted"`, [3]int{0, 1, 2}, false, func(r record) string { return r.Reason + ":accepted" }},
		{"second_copy", "let copy = input0\nlet other = copy\nother.title = \"ignored\"\ncopy.title = \"draft\"\ncopy.state = \"wait\"\ncopy.reason = \"deferred\"\nreturn Candidate{title: copy.title, state: copy.state, reason: copy.reason + other.reason}", `other.state + ":" + copy.title`, [3]int{1, 2, 3}, false, func(r record) string { return r.State + ":" + r.Title + r.Reason }},
	}
}
func fixture(s shape, language string, budget int) ([]byte, []byte) {
	const header = `package records
namespace fieldupdates
entity Text id "fieldupdates://text"
entity Boolean id "fieldupdates://boolean"
entity Candidate id "fieldupdates://candidate" fields {
 field title id "fieldupdates://candidate/title" type string required one
 field state id "fieldupdates://candidate/state" type string required one
 field reason id "fieldupdates://candidate/reason" type string required one
}
activity Select(Candidate, Boolean) -> Candidate computes `
	intents := [3]string{"Keep the original title.", "Set state to ready.", "Use the declared expression with the current and saved values."}
	if language == "ko" {
		intents = [3]string{"원래 제목을 사용한다.", "상태를 ready로 만든다.", "현재 값과 저장한 값으로 지정된 표현식을 사용한다."}
	}
	var b strings.Builder
	b.WriteString(header)
	b.WriteByte('`')
	b.WriteString(s.body)
	b.WriteString("` assembling {\n")
	for i, expression := range [3]string{"input0.title", `"ready"`, s.reason} {
		fmt.Fprintf(&b, "choice %q field_update at %q alternative %s intent %s\n", []string{"title", "state", "reason"}[i], strconv.Itoa(s.index[i]), strconv.Quote(expression), strconv.Quote(intents[i]))
	}
	inputs := []sample{{record{"한글", "queued", "검토"}, true}, {record{"", "waiting", ""}, true}, {record{"later", "queued", "retained"}, false}, {record{"already", "ready", "done"}, true}}
	var cases []any
	for _, c := range inputs {
		wanted := record{c.input.Title, "ready", s.wanted(c.input)}
		if s.guarded && (!c.enabled || c.input.State == "ready") {
			wanted = c.input
		}
		fmt.Fprintf(&b, "value_case %s -> %s\n", strconv.Quote(string(encode([]any{c.input, c.enabled}))), strconv.Quote(string(encode(wanted))))
		cases = append(cases, map[string]any{"inputs": map[string]any{"Select.input0": c.input, "Select.input1": c.enabled}, "expected": map[string]any{"Select": wanted, "Label": wanted.Title + ":" + wanted.State + ":" + wanted.Reason}})
	}
	fmt.Fprintf(&b, "attempts %q\n}\n", strconv.Itoa(budget))
	b.WriteString("activity Label(Candidate) -> Text computes `return input.title + \":\" + input.state + \":\" + input.reason`\nbind Select.result -> Label.input\n")
	return []byte(b.String()), encode(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": cases})
}
