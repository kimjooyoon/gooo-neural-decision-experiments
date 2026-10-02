package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	stdhash "hash"
	"io"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const rawCap = 768 << 20
const lineCap = 1 << 20

var privateText = regexp.MustCompile(`/Users/|/home/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)

func hash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

func save(name string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(name, append(raw, '\n'), 0644)
}

type journal struct {
	file  *os.File
	hash  stdhash.Hash
	bytes int64
	lines int
	used  *int64
}

func openJournal(name string, used *int64) (*journal, error) {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, err
	}
	return &journal{file: f, hash: sha256.New(), used: used}, nil
}

func (j *journal) append(value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(raw)+1 > lineCap || *j.used+int64(len(raw)+1) > rawCap {
		return errors.New("raw evidence cap reached; retain prefix and stop")
	}
	if privateText.Match(raw) {
		return errors.New("private text in attempted public journal")
	}
	raw = append(raw, '\n')
	n, err := j.file.Write(raw)
	_, _ = j.hash.Write(raw[:n])
	j.bytes += int64(n)
	*j.used += int64(n)
	if err == nil && n != len(raw) {
		err = io.ErrShortWrite
	}
	if err == nil {
		j.lines++
	}
	return err
}

func (j *journal) pin() map[string]any {
	return map[string]any{"sha256": hex.EncodeToString(j.hash.Sum(nil)), "bytes": j.bytes, "lines": j.lines}
}

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(raw []byte) (int, error) {
	if b.Len()+len(raw) > lineCap {
		return 0, errors.New("native child output exceeds cap")
	}
	return b.Buffer.Write(raw)
}

func child(workspace, binary string) ([]byte, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "body-context", "--plan", "plan.json", "--activity", "ChoosePath",
		"--feature-version", decision.SemanticContextIntentFeatureVersion, "source.gooo")
	command.Dir = workspace
	var stdout, stderr limitedBuffer
	command.Stdout, command.Stderr = &stdout, &stderr
	start := time.Now()
	err := command.Run()
	return stdout.Bytes(), time.Since(start).Nanoseconds(), err
}
