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
	raw, err := io.ReadAll(io.LimitReader(r, 1<<20+1))
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 {
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
