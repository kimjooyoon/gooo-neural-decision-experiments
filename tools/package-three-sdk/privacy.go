package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

var privateText = regexp.MustCompile(`/Users/|/home/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)

func publicJSON(raw []byte) error {
	if err := decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	var walk func(any) error
	walk = func(v any) error {
		switch x := v.(type) {
		case string:
			if privateText.MatchString(x) {
				return errors.New("decoded private text rejected")
			}
		case []any:
			for _, y := range x {
				if err := walk(y); err != nil {
					return err
				}
			}
		case map[string]any:
			for k, y := range x {
				if privateText.MatchString(k) {
					return errors.New("private JSON key rejected")
				}
				if err := walk(y); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(v)
}
func privacy(r io.Reader, name string) error {
	if strings.HasSuffix(name, ".jsonl") {
		s := bufio.NewScanner(r)
		s.Buffer(make([]byte, 32768), 1<<20)
		for s.Scan() {
			if err := publicJSON(s.Bytes()); err != nil {
				return err
			}
		}
		return s.Err()
	}
	limit := int64(1 << 20)
	// The exact frozen list of 2,485 missing identities is metadata, not a
	// session capture. Preserve it whole under a separate explicit 4-MiB bound.
	if name == tailPhase+"/preexecution.json" {
		limit = 4 << 20
	}
	raw, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return err
	}
	if int64(len(raw)) > limit {
		return errors.New("bounded public metadata required")
	}
	if strings.HasSuffix(name, ".json") {
		return publicJSON(raw)
	}
	if privateText.Match(raw) {
		return errors.New("private text rejected")
	}
	return nil
}
