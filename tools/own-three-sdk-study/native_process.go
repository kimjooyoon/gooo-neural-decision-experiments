package main

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

type childBuffer struct {
	bytes.Buffer
	Cap int
}

func (b *childBuffer) Write(raw []byte) (int, error) {
	if b.Len()+len(raw) > b.Cap {
		return 0, errors.New("bounded child output exceeded; prior bytes retained")
	}
	return b.Buffer.Write(raw)
}
func child(dir, binary string, args ...string) ([]byte, string, childMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return childWithContext(ctx, dir, binary, args...)
}
func childWithContext(ctx context.Context, dir, binary string, args ...string) ([]byte, string, childMetrics, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir, cmd.WaitDelay = dir, time.Second
	cmd.Env = append(os.Environ(), "GOOO_LAYA_URL=", "GOOO_LAYA_API_KEY=", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	configureChild(cmd)
	stdout, stderr := childBuffer{Cap: lineCap}, childBuffer{Cap: 128 << 10}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	start := time.Now()
	err := cmd.Run()
	m := childMetrics{Wall: time.Since(start).Nanoseconds(), Started: cmd.Process != nil}
	if cmd.ProcessState != nil {
		m.User, m.System = cmd.ProcessState.UserTime().Nanoseconds(), cmd.ProcessState.SystemTime().Nanoseconds()
		m.RSS = childRSS(cmd.ProcessState)
		m.CPU = 100 * float64(m.User+m.System) / float64(m.Wall)
	}
	if err != nil || stderr.Len() != 0 {
		return stdout.Bytes(), stderr.String(), m, errors.Join(errors.New("bounded child failed; captured stdout/stderr retained; overflowing output may be partial"), ctx.Err())
	}
	return stdout.Bytes(), stderr.String(), m, nil
}
func validChild(m childMetrics) bool {
	return m.Started && m.Wall > 0 && m.User >= 0 && m.System >= 0 && m.RSS > 0 && m.CPU == 100*float64(m.User+m.System)/float64(m.Wall)
}
func compilerPins(binary, goBinary string) (threestudent.Pin, threestudent.Pin, threestudent.Pin, error) {
	var empty threestudent.Pin
	worker := filepath.Join(filepath.Dir(binary), "gooo-body-worker")
	for _, path := range []string{binary, worker} {
		info, err := buildinfo.ReadFile(path)
		if err != nil || info.GoVersion != "go1.27.1" {
			return empty, empty, empty, errors.New("exact native/worker Go 1.27.1 required")
		}
		settings := map[string]string{}
		for _, s := range info.Settings {
			settings[s.Key] = s.Value
		}
		sdk := ""
		for _, d := range info.Deps {
			if d.Path == "github.com/kimjooyoon/gooo-decision-runtime" && d.Replace == nil {
				sdk = d.Version
			}
		}
		if settings["vcs.revision"] != nativeRevision || settings["vcs.modified"] != "false" || sdk != nativeSDK {
			return empty, empty, empty, errors.New("clean adopted main and released SDK.13 required")
		}
	}
	info, err := buildinfo.ReadFile(goBinary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return empty, empty, empty, errors.New("exact emitted-Go execution compiler required")
	}
	b, err := threestudent.FilePin(binary)
	if err != nil {
		return empty, empty, empty, err
	}
	w, err := threestudent.FilePin(worker)
	if err != nil {
		return empty, empty, empty, err
	}
	g, err := threestudent.FilePin(goBinary)
	return b, w, g, err
}
func executeGo(binary, source string, cases []pathplan.TestCase) ([]int64, string, string, childMetrics, error) {
	dir, err := os.MkdirTemp("", "gooo-three-compiled-execute-")
	if err != nil {
		return nil, "", "", childMetrics{}, err
	}
	defer os.RemoveAll(dir)
	if err = os.Mkdir(filepath.Join(dir, "projection"), 0700); err != nil {
		return nil, "", "", childMetrics{}, err
	}
	var caller strings.Builder
	caller.WriteString("package main\nimport(\"encoding/json\";\"os\";p \"gooo.three.execution/projection\")\nfunc main(){json.NewEncoder(os.Stdout).Encode([]int64{")
	for _, c := range cases {
		fmt.Fprintf(&caller, "p.ChoosePath(%d),", c.Input)
	}
	caller.WriteString("})}\n")
	for name, text := range map[string]string{"go.mod": "module gooo.three.execution\n\ngo 1.27.1\n", "projection/generated.go": source, "main.go": caller.String()} {
		if err = os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			return nil, "", "", childMetrics{}, err
		}
	}
	raw, stderr, m, err := child(dir, binary, "run", ".")
	if !utf8.Valid(raw) || !utf8.ValidString(stderr) {
		return nil, "", stderr, m, errors.New("non-UTF8 Go output cannot be silently converted")
	}
	if err != nil {
		return nil, string(raw), stderr, m, err
	}
	var values []int64
	if err = json.Unmarshal(raw, &values); err != nil || len(values) != len(cases) {
		return values, string(raw), stderr, m, errors.New("compiled Go ordered count differs")
	}
	for i, a := range values {
		if a != cases[i].Expected {
			return values, string(raw), stderr, m, errors.New("compiled Go ordered independent oracle differs")
		}
	}
	return values, string(raw), stderr, m, nil
}
