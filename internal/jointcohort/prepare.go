package jointcohort

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func prepare(pair [2]*sourceRow) (View, error) {
	if pair[0] == nil || pair[1] == nil {
		return View{}, errors.New("complete source pair required")
	}
	a, b := pair[0], pair[1]
	if a.Group != b.Group || a.Language != b.Language || a.Config != b.Config || a.Goal != b.Goal || a.Family != b.Family || a.Joint != b.Joint || a.SourceSHA != b.SourceSHA || !reflect.DeepEqual(a.Target, b.Target) {
		return View{}, errors.New("source pair identity differs")
	}
	plan, err := jointcompositionstudy.Fixture(a.Family, a.Config, a.Goal, a.Language)
	if err != nil {
		return View{}, err
	}
	target, err := jointcompositionstudy.Target(plan, a.Family, a.Config, a.Goal)
	if err != nil || !reflect.DeepEqual(target, a.Target) || a.Split != jointcompositionstudy.Split(a.Config) || b.Split != a.Split {
		return View{}, errors.New("independent oracle target differs")
	}
	original, err := pathplan.Prepare(plan)
	if err != nil {
		return View{}, err
	}
	body, err := json.Marshal(original.Fallback().GoooBody())
	if err != nil {
		return View{}, err
	}
	source := []byte("package freshcomposition\nnamespace freshcomposition\nentity Integer id \"freshcomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	if a.SourceSHA != SHA(source) {
		return View{}, errors.New("native original source hash differs")
	}
	for i, r := range pair {
		choice := plan.Decisions[i]
		natural := choice.Intent[strings.LastIndex(choice.Intent, "intent: ")+8:]
		fields, e := original.SourceFeatures(choice.ID)
		if e != nil {
			return View{}, e
		}
		text, e := decision.EncodeSemanticContextInput(fields, natural)
		if e != nil || r.Input.Text != text || r.Input.ID != choice.ID || r.Input.SHA != "sha256:"+SHA([]byte(text)) || r.Input.Natural != "sha256:"+SHA([]byte(natural)) || r.Options != [2]string{choice.Options[0].Label, choice.Options[1].Label} || r.Marginals != target.Marginals[i] {
			return View{}, errors.New("complete source-bound input differs")
		}
		plan.Decisions[i].Intent = text
	}
	joint, err := jointdecision.Encode([2]string{pair[0].Input.Text, pair[1].Input.Text})
	if err != nil || joint != a.Joint || a.JointSHA != SHA([]byte(joint)) || b.JointSHA != a.JointSHA {
		return View{}, errors.New("joint ordered input differs")
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return View{}, err
	}
	cases, err := jointcompositionstudy.Cases(a.Family, a.Config, a.Goal)
	if err != nil {
		return View{}, err
	}
	return View{a.Group + "/" + a.Language, a.Group, a.Family, a.Config, a.Goal, a.Language, a.Split, a.SourceSHA, joint, target, prepared, cases}, nil
}
