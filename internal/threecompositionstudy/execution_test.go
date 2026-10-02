package threecompositionstudy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

// The ordinary Go compiler executes every mask in two configurations, including
// both <= and == comparisons. This is separate from the typed interpreter.
func TestIndependentCompiledGoAllFamiliesAndMasks(t *testing.T) {
	var source, main strings.Builder
	source.WriteString("package main\nimport \"fmt\"\n")
	main.WriteString("func main() {\n")
	functions, invocations := 0, 0
	for _, family := range Families {
		for config := range 2 {
			plan, err := Fixture(family, config, 0, "en")
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := pathplan.Prepare(plan)
			if err != nil {
				t.Fatal(err)
			}
			for mask := range 8 {
				choices, err := Choices(plan, mask)
				if err != nil {
					t.Fatal(err)
				}
				program, err := prepared.Compile(choices)
				if err != nil {
					t.Fatal(err)
				}
				name := fmt.Sprintf("Fixture%d", functions)
				body := strings.TrimPrefix(program.GoSource(), "package generated\n")
				if !strings.Contains(body, "func ChoosePath(") || strings.HasPrefix(body, "package ") {
					t.Fatal("closed generated source contract changed")
				}
				body = strings.Replace(body, "func ChoosePath(", "func "+name+"(", 1)
				source.WriteString(body)
				cases, err := Cases(family, config, mask)
				if err != nil {
					t.Fatal(err)
				}
				for _, test := range cases {
					fmt.Fprintf(&main, "if got := %s(%d); got != %d { panic(fmt.Sprintf(\"fixture=%d input=%d got=%%d\", got)) }\n",
						name, test.Input, test.Expected, functions, test.Input)
					invocations++
				}
				functions++
			}
		}
	}
	fmt.Fprintf(&main, "fmt.Println(\"PASS functions=%d ordered_invocations=%d\")\n}\n", functions, invocations)
	source.WriteString(main.String())
	if functions != 128 || invocations != 2048 {
		t.Fatal("compiled execution denominator", functions, invocations)
	}
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "main.go"), []byte(source.String()), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "run", "main.go")
	command.Dir = workspace
	command.Env = append(os.Environ(), "GOTOOLCHAIN=go1.27.1")
	output, err := command.CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "PASS functions=128 ordered_invocations=2048" {
		t.Fatalf("independent compiled execution failed: %v %s", err, output)
	}
	t.Log(strings.TrimSpace(string(output)))
}
