package main

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

func nativeViews(views []threecohort.View) ([]threecohort.View, error) {
	selected := make([]threecohort.View, 0, 128)
	for _, v := range views {
		if v.Config == 20 && v.Split == "development" {
			selected = append(selected, v)
		}
	}
	if len(selected) != 128 {
		return nil, errors.New("exact original development configuration 20 requires 128 views")
	}
	return selected, nil
}

type nativePrior struct {
	Attempts  []pathplan.SearchAttempt
	Proposals map[string]string
	Calls     int
}

func sdkNativePriors(prefix, tail string, views []threecohort.View) (map[string]map[string]nativePrior, error) {
	wanted := map[string]bool{}
	for _, v := range views {
		wanted[v.ID] = true
	}
	all := map[string]map[string]nativePrior{}
	for _, policy := range nativePolicies {
		id := policy.Candidate
		if all[id] != nil {
			continue
		}
		all[id] = map[string]nativePrior{}
		for _, dir := range []string{prefix, tail} {
			f, err := os.Open(filepath.Join(dir, filename("development", id)))
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, err
			}
			s := bufio.NewScanner(f)
			s.Buffer(make([]byte, 32768), lineCap)
			for s.Scan() {
				var o observation
				if err = threecohort.Decode(s.Bytes(), &o); err != nil {
					f.Close()
					return nil, err
				}
				if o.Policy != id {
					f.Close()
					return nil, errors.New("frozen SDK policy differs")
				}
				if wanted[o.Capture.ViewID] {
					if _, ok := all[id][o.Capture.ViewID]; ok {
						f.Close()
						return nil, errors.New("duplicate frozen native SDK prior")
					}
					all[id][o.Capture.ViewID] = priorOf(o.Capture)
				}
			}
			err = s.Err()
			f.Close()
			if err != nil {
				return nil, err
			}
		}
		if len(all[id]) != 128 {
			return nil, errors.New("complete 128-view SDK native prior required")
		}
	}
	return all, nil
}
func priorOf(c threefeedback.Capture) nativePrior {
	return nativePrior{c.Search.Attempts, c.Search.InitialProposals, c.Search.Selection.ModelCalls}
}
