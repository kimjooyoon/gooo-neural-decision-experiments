// Package bilingualstudy pairs source-bound Gooo views without inventing labels.
package bilingualstudy

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/feedbackstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

const DatasetSHA = "570d1d73bfea32662b869e5cfbf77e6d04df838b703191b076be58c1d2180dbf"

type Pair struct {
	ID       string               `json:"id"`
	Program  string               `json:"program_id"`
	Family   string               `json:"family"`
	Template string               `json:"template_pair_id"`
	Split    string               `json:"split"`
	Rows     [2]feedbackstudy.Row `json:"-"` // English, Korean; fixed two-row inference layout.
	RowIDs   [2]string            `json:"row_ids"`
	Hashes   [2]string            `json:"input_sha256"`
	Options  [2]string            `json:"eligible_labels"`
	Targets  [2]float32           `json:"finite_soft_targets"`
}

func Load(filename string) ([]Pair, error) {
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 16<<20 {
		return nil, errors.New("bounded regular frozen dataset required")
	}
	raw, err := os.ReadFile(filename)
	if err != nil || feedbackstudy.Hash(raw) != DatasetSHA {
		return nil, errors.New("frozen feedback dataset digest differs")
	}
	var rows []feedbackstudy.Row
	s := bufio.NewScanner(bytes.NewReader(raw))
	s.Buffer(make([]byte, 4096), 16<<10)
	for s.Scan() {
		var row feedbackstudy.Row
		if err := strictjson.Decode(s.Bytes(), &row); err != nil {
			return nil, err
		}
		if row.View == "gooo" {
			original := feedbackstudy.Original{ID: strings.TrimSuffix(row.ID, "-feedback"),
				InstructionID: row.InstructionID, ProgramID: row.ProgramID, TemplateID: row.TemplateID,
				ConfigurationID: row.ConfigurationID, ConfigurationIndex: row.Configuration,
				Family: row.Family, Language: row.Language, Split: row.Split, View: row.View,
				Text: row.OriginalText, Label: row.IntentionLabel}
			derived, err := feedbackstudy.Derive(original)
			if err != nil || !reflect.DeepEqual(row, derived) {
				return nil, errors.New("source-bound typed finite target differs")
			}
			rows = append(rows, row)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	pairs, err := Group(rows)
	if err == nil && len(pairs) != 1040 {
		err = errors.New("1040 reused bilingual Gooo pairs required")
	}
	return pairs, err
}

func Group(rows []feedbackstudy.Row) ([]Pair, error) {
	groups := make(map[string]Pair)
	for _, row := range rows {
		if row.View != "gooo" || !row.Eligible || (row.Language != "en" && row.Language != "ko") {
			return nil, errors.New("bounded Gooo English/Korean row required")
		}
		template, err := strconv.Atoi(strings.TrimPrefix(row.TemplateID, row.Family+"-t"))
		if err != nil || template < 0 || template > 7 || (template%2 == 0) != (row.Language == "en") {
			return nil, errors.New("explicit adjacent bilingual template required")
		}
		key := fmt.Sprintf("%s-gooo-p%d", row.ProgramID, template/2)
		p := groups[key]
		language := template % 2
		if p.Rows[language].ID != "" {
			return nil, errors.New("duplicate language in pair")
		}
		p.ID, p.Program, p.Family, p.Split = key, row.ProgramID, row.Family, row.Split
		p.Template = fmt.Sprintf("%s-p%d", row.Family, template/2)
		p.Rows[language], p.RowIDs[language], p.Hashes[language] = row, row.ID, row.InputSHA
		groups[key] = p
	}
	result := make([]Pair, 0, len(groups))
	for _, p := range groups {
		a, b := p.Rows[0], p.Rows[1]
		if a.ID == "" || b.ID == "" || a.Split != b.Split || a.ProgramID != b.ProgramID ||
			a.IntentionLabel != b.IntentionLabel || a.Options != b.Options ||
			!reflect.DeepEqual(a.Cases, b.Cases) || !reflect.DeepEqual(a.Accepted, b.Accepted) ||
			a.OptionPassed != b.OptionPassed || len(a.Accepted) < 1 || len(a.Accepted) > 2 {
			return nil, errors.New("pair semantics, finite contract or split differs")
		}
		// Both source views show the same fallback Gooo body before their intent.
		if strings.Split(a.OriginalText, "\nintent: ")[0] != strings.Split(b.OriginalText, "\nintent: ")[0] {
			return nil, errors.New("bilingual Gooo structure differs")
		}
		p.Options = a.Options
		for i, option := range p.Options {
			for _, accepted := range a.Accepted {
				if option == accepted {
					p.Targets[i] = 1 / float32(len(a.Accepted))
				}
			}
		}
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
