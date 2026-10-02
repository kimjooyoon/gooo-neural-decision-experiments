package main

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const nativeDeployed = "363a3d8aa365c35dd634c241248b444de0050973"

type metrics struct {
	Wall   int64   `json:"wall_ns"`
	User   int64   `json:"user_cpu_ns"`
	System int64   `json:"system_cpu_ns"`
	RSS    int64   `json:"child_max_rss_bytes"`
	CPU    float64 `json:"process_cpu_percent_of_one_core_over_wall"`
}
type bounded struct{ bytes.Buffer }

func (b *bounded) Write(raw []byte) (int, error) {
	if b.Len()+len(raw) > 1<<20 {
		return 0, errors.New("child byte bound exceeded")
	}
	return b.Buffer.Write(raw)
}
func child(dir, binary string, args ...string) ([]byte, metrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
	cmd.Env = append(os.Environ(), "GOOO_LAYA_URL=", "GOOO_LAYA_API_KEY=", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	configure(cmd)
	var stdout, stderr bounded
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	start := time.Now()
	err := cmd.Run()
	m := metrics{Wall: time.Since(start).Nanoseconds()}
	if cmd.ProcessState != nil {
		m.User, m.System = cmd.ProcessState.UserTime().Nanoseconds(), cmd.ProcessState.SystemTime().Nanoseconds()
		m.RSS = maxRSS(cmd.ProcessState)
		m.CPU = 100 * float64(m.User+m.System) / float64(m.Wall)
	}
	if err != nil || stderr.Len() != 0 {
		return stdout.Bytes(), m, errors.New("bounded child failed; stdout retained")
	}
	return stdout.Bytes(), m, nil
}

func nativePins(binary, goBinary string) error {
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("exact native Go 1.27.1 required")
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	sdk := ""
	for _, d := range info.Deps {
		if d.Path == "github.com/kimjooyoon/gooo-decision-runtime" {
			sdk = d.Version
		}
	}
	if settings["vcs.revision"] != nativeDeployed || settings["vcs.modified"] != "false" || sdk != "v0.2.12-experimental" {
		return errors.New("clean native main and SDK.12 required")
	}
	info, err = buildinfo.ReadFile(goBinary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("exact emitted-Go execution compiler required")
	}
	return nil
}

func executeGo(goBinary, source string, cases []pathplan.TestCase) ([]int64, metrics, error) {
	dir, err := os.MkdirTemp("", "gooo-fresh-execute-")
	if err != nil {
		return nil, metrics{}, err
	}
	defer os.RemoveAll(dir)
	if err = os.Mkdir(filepath.Join(dir, "projection"), 0700); err != nil {
		return nil, metrics{}, err
	}
	var caller strings.Builder
	caller.WriteString("package main\nimport(\"encoding/json\";\"os\";p \"gooo.fresh.execution/projection\")\nfunc main(){json.NewEncoder(os.Stdout).Encode([]int64{")
	for _, c := range cases {
		fmt.Fprintf(&caller, "p.ChoosePath(%d),", c.Input)
	}
	caller.WriteString("})}\n")
	for name, content := range map[string]string{"go.mod": "module gooo.fresh.execution\n\ngo 1.27.1\n", "projection/generated.go": source, "main.go": caller.String()} {
		if err = os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			return nil, metrics{}, err
		}
	}
	raw, m, err := child(dir, goBinary, "run", ".")
	if err != nil {
		return nil, m, err
	}
	var values []int64
	if err = json.Unmarshal(raw, &values); err != nil || len(values) != len(cases) {
		return nil, m, errors.New("compiled Go actual count differs")
	}
	for i, v := range values {
		if v != cases[i].Expected {
			return nil, m, errors.New("compiled Go independent oracle differs")
		}
	}
	return values, m, nil
}
