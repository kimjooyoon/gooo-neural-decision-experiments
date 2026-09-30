// Command generate emits a frozen, model-free cohort of typed body plans.
// Its mathematical references are written independently from the runtime
// body-plan evaluator so expected values do not come from compiler output.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const planSchema = "gooo/typed-body-plan/v1"

type Plan struct {
	Schema      string       `json:"schema"`
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	ResultType  string       `json:"result_type"`
	Expressions []Expression `json:"expressions"`
	Statements  []Statement  `json:"statements"`
	Root        []int        `json:"root"`
}

type Expression struct {
	Kind      string   `json:"kind"`
	Name      string   `json:"name,omitempty"`
	Int       *int64   `json:"int,omitempty"`
	Bool      *bool    `json:"bool,omitempty"`
	Operation string   `json:"operation,omitempty"`
	Left      *int     `json:"left,omitempty"`
	Right     *int     `json:"right,omitempty"`
	HoleID    string   `json:"hole_id,omitempty"`
	Text      string   `json:"text,omitempty"`
	Allowed   []string `json:"allowed,omitempty"`
	Fallback  string   `json:"fallback,omitempty"`
}

type Statement struct {
	Kind string `json:"kind"`
	Name string `json:"name,omitempty"`
	Expr *int   `json:"expr,omitempty"`
	Then []int  `json:"then,omitempty"`
	Else []int  `json:"else,omitempty"`
}

type Expected struct {
	Type string `json:"type"`
	Int  *int64 `json:"int,omitempty"`
	Bool *bool  `json:"bool,omitempty"`
}

type Case struct {
	Input    int64    `json:"input"`
	Expected Expected `json:"expected"`
}

type Row struct {
	ID               string            `json:"id"`
	Family           string            `json:"family"`
	Plan             Plan              `json:"plan"`
	OracleOperations map[string]string `json:"oracle_operations"`
	TrainingCases    []Case            `json:"training_cases"`
	HeldoutCases     []Case            `json:"heldout_cases"`
}

type Value struct {
	typ string
	i   int64
	b   bool
}

type Builder struct {
	plan Plan
}

type Scenario struct {
	family string
	build  func(*Builder, int) (map[string]string, func(int64) Value, []int64, error)
}

var intOps = []string{"add", "multiply", "subtract"}
var cmpOps = []string{"equal", "less_equal", "less_than"}
var boolOps = []string{"and", "equal", "or"}

var coreTrainInputs = []int64{
	math.MinInt64, math.MinInt64 + 1, -1000003, -1024, -257, -17, -7, -5, -3, -2, -1,
	0, 1, 2, 3, 5, 7, 17, 257, 1024, 1000003, math.MaxInt64 - 1, math.MaxInt64,
}

var thresholdInputs = []int64{-17, -7, -3, -1, 0, 1, 3, 7, 17}

var negativeHeldoutPool = []int64{math.MinInt64 + 2, math.MinInt64 + 3, math.MinInt64 + 19, -9000000000000000001, -9000000000000000000, -1000002, -65537, -4097, -997, -43, -42, -41, -13, -12, -11, -9, -6, -4}
var positiveHeldoutPool = []int64{math.MaxInt64 - 2, math.MaxInt64 - 3, math.MaxInt64 - 19, 9000000000000000000, 9000000000000000001, 1000002, 65537, 4097, 997, 43, 42, 41, 13, 12, 11, 9, 6, 4}

func ptrInt(value int) *int     { return &value }
func ptrI64(value int64) *int64 { return &value }
func ptrBool(value bool) *bool  { return &value }

func newBuilder(index int) *Builder {
	id := fmt.Sprintf("body-%03d", index)
	return &Builder{plan: Plan{Schema: planSchema, ID: id, Name: fmt.Sprintf("Body%03d", index)}}
}

func (b *Builder) input() int              { return b.add(Expression{Kind: "input", Name: "input"}) }
func (b *Builder) integer(value int64) int { return b.add(Expression{Kind: "int", Int: ptrI64(value)}) }
func (b *Builder) boolean(value bool) int {
	return b.add(Expression{Kind: "bool", Bool: ptrBool(value)})
}
func (b *Builder) local(name string) int { return b.add(Expression{Kind: "local", Name: name}) }

func (b *Builder) binary(operation string, left, right int) int {
	return b.add(Expression{Kind: "binary", Operation: operation, Left: ptrInt(left), Right: ptrInt(right)})
}

func (b *Builder) hole(id, text string, left, right int, allowed []string, fallback string) int {
	return b.add(Expression{Kind: "hole", HoleID: id, Text: text, Left: ptrInt(left), Right: ptrInt(right), Allowed: append([]string(nil), allowed...), Fallback: fallback})
}

func (b *Builder) add(expression Expression) int {
	index := len(b.plan.Expressions)
	b.plan.Expressions = append(b.plan.Expressions, expression)
	return index
}

func (b *Builder) let(name string, expr int) int {
	return b.statement(Statement{Kind: "let", Name: name, Expr: ptrInt(expr)})
}
func (b *Builder) assign(name string, expr int) int {
	return b.statement(Statement{Kind: "assign", Name: name, Expr: ptrInt(expr)})
}
func (b *Builder) ret(expr int) int {
	return b.statement(Statement{Kind: "return", Expr: ptrInt(expr)})
}
func (b *Builder) branch(expr int, then, otherwise []int) int {
	return b.statement(Statement{Kind: "if", Expr: ptrInt(expr), Then: append([]int(nil), then...), Else: append([]int(nil), otherwise...)})
}

func (b *Builder) statement(statement Statement) int {
	index := len(b.plan.Statements)
	b.plan.Statements = append(b.plan.Statements, statement)
	return index
}

func (b *Builder) root(statements ...int) { b.plan.Root = append([]int(nil), statements...) }

func run() error {
	if len(os.Args) > 2 {
		return errors.New("usage: generate [empty-output-directory]")
	}
	rows, err := buildRows()
	if err != nil {
		return err
	}
	destination := outputDir()
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	if err := validateOutputDirectory(destination); err != nil {
		return err
	}
	jsonl, err := encodeRows(rows)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(destination, "cohort.jsonl"), jsonl, 0o644); err != nil {
		return err
	}
	manifest, err := makeManifest(rows, jsonl)
	if err != nil {
		return err
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	manifestJSON = append(manifestJSON, '\n')
	if err := os.WriteFile(filepath.Join(destination, "manifest.json"), manifestJSON, 0o644); err != nil {
		return err
	}
	return nil
}

func outputDir() string {
	if len(os.Args) > 1 && os.Args[1] != "" {
		absolute, err := filepath.Abs(os.Args[1])
		if err == nil {
			return absolute
		}
	}
	source, err := sourcePath()
	if err != nil {
		return "studies/body-plan-v1"
	}
	return filepath.Dir(source)
}

func sourcePath() (string, error) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("cannot locate generator source")
	}
	return filepath.Abs(source)
}

func validateOutputDirectory(destination string) error {
	source, err := sourcePath()
	if err != nil {
		return err
	}
	if destination == filepath.Dir(source) {
		for _, name := range []string{"cohort.jsonl", "manifest.json"} {
			if _, err := os.Lstat(filepath.Join(destination, name)); err == nil {
				return fmt.Errorf("refusing to overwrite existing %s", name)
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		return nil
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return errors.New("output directory must be empty")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildRows() ([]Row, error) {
	scenarios := []Scenario{
		{family: "arithmetic-pipeline", build: buildArithmetic},
		{family: "threshold-comparison", build: buildComparison},
		{family: "boolean-composition", build: buildBoolean},
		{family: "reassignment", build: buildReassignment},
		{family: "piecewise-branch", build: buildPiecewise},
		{family: "nested-branch", build: buildNested},
		{family: "polynomial-order", build: buildPolynomial},
		{family: "signed-boundary", build: buildBoundary},
	}
	rows := make([]Row, 0, len(scenarios)*16)
	seen := make(map[string]bool)
	for familyIndex, scenario := range scenarios {
		for variant := 0; variant < 16; variant++ {
			index := familyIndex*16 + variant + 1
			b := newBuilder(index)
			ops, oracle, critical, err := scenario.build(b, variant)
			if err != nil {
				return nil, fmt.Errorf("%s variant %d: %w", scenario.family, variant, err)
			}
			b.plan.ResultType = oracle(0).typ
			b.plan.ID = fmt.Sprintf("body-%03d", index)
			b.plan.Name = fmt.Sprintf("Body%03d", index)
			trainInputs := makeTrainInputs(critical)
			heldoutInputs := makeHeldoutInputs(index, trainInputs, b.plan)
			row := Row{ID: b.plan.ID, Family: scenario.family, Plan: b.plan, OracleOperations: ops}
			for _, input := range trainInputs {
				row.TrainingCases = append(row.TrainingCases, Case{Input: input, Expected: expected(oracle(input))})
			}
			for _, input := range heldoutInputs {
				row.HeldoutCases = append(row.HeldoutCases, Case{Input: input, Expected: expected(oracle(input))})
			}
			if err := validateRow(row); err != nil {
				return nil, fmt.Errorf("%s: %w", row.ID, err)
			}
			if err := validateOracleAgreement(row, oracle); err != nil {
				return nil, fmt.Errorf("%s: %w", row.ID, err)
			}
			if err := validateGoldDiscrimination(row); err != nil {
				return nil, fmt.Errorf("%s: %w", row.ID, err)
			}
			key := canonicalIntent(row)
			if seen[key] {
				return nil, fmt.Errorf("duplicate canonical intent at %s", row.ID)
			}
			seen[key] = true
			rows = append(rows, row)
		}
	}
	if len(rows) < 100 || len(seen) != len(rows) {
		return nil, errors.New("cohort is short of the distinct-intent requirement")
	}
	return rows, nil
}

func buildArithmetic(b *Builder, v int) (map[string]string, func(int64) Value, []int64, error) {
	x := b.input()
	c, d := int64((v%7)+2), int64((v%5)+3)
	goldA, goldB := cycle(intOps, v), cycle(intOps, v+1)
	fallbackA, fallbackB := nextAllowed(intOps, goldA, v+1), nextAllowed(intOps, goldB, v+2)
	holeA := "h1"
	var holeB string
	var result int
	var oracle func(int64) Value
	switch v % 4 {
	case 0:
		base := b.binary("add", x, b.integer(c))
		result = b.hole(holeA, textFor(goldA, false, v), base, b.integer(d), intOps, fallbackA)
		oracle = func(input int64) Value { return intValue(applyInt(goldA, input+c, d)) }
	case 1:
		first := b.hole(holeA, textFor(goldA, false, v), x, b.integer(c), intOps, fallbackA)
		result = b.binary("subtract", first, b.integer(d))
		oracle = func(input int64) Value { return intValue(applyInt(goldA, input, c) - d) }
	case 2:
		first := b.hole(holeA, textFor(goldA, false, v), x, b.integer(c), intOps, fallbackA)
		holeB = "h2"
		result = b.hole(holeB, textFor(goldB, true, v), first, b.integer(d), intOps, fallbackB)
		oracle = func(input int64) Value { return intValue(applyInt(goldB, applyInt(goldA, input, c), d)) }
	case 3:
		base := b.binary("multiply", x, b.integer(d))
		result = b.hole(holeA, textFor(goldA, false, v), b.integer(c), base, intOps, fallbackA)
		oracle = func(input int64) Value { return intValue(applyInt(goldA, c, input*d)) }
	}
	b.root(b.ret(result))
	ops := map[string]string{holeA: goldA}
	if holeB != "" {
		ops[holeB] = goldB
	}
	return ops, oracle, []int64{c, d}, nil
}

func buildComparison(b *Builder, v int) (map[string]string, func(int64) Value, []int64, error) {
	x := b.input()
	c := int64(v%9 - 4)
	target := int64(v*2 - 15)
	gold := cycle(cmpOps, v)
	base := x
	var critical []int64
	switch v % 4 {
	case 0:
		base = x
		critical = append(critical, target)
	case 1:
		base = b.binary("add", x, b.integer(c))
		critical = append(critical, target-c)
	case 2:
		probe := int64(v%5 - 2)
		if c+2 == 0 {
			c = -1
		}
		base = b.binary("multiply", x, b.integer(c+2))
		target = probe * (c + 2)
		critical = append(critical, probe)
		if c+2 != 0 {
			critical = append(critical, target/(c+2), target/(c+2)+1, target/(c+2)-1)
		}
	case 3:
		adjusted := b.binary("subtract", x, b.integer(c))
		base = adjusted
		critical = append(critical, target+c)
	}
	var rootPrefix []int
	if v%4 == 3 {
		rootPrefix = append(rootPrefix, b.let("measure", base))
		base = b.local("measure")
	}
	h := b.hole("h1", compareText(gold, v), base, b.integer(target), cmpOps, nextAllowed(cmpOps, gold, v+1))
	b.root(append(rootPrefix, b.ret(h))...)
	return map[string]string{"h1": gold}, func(input int64) Value {
		var value int64
		switch v % 4 {
		case 0:
			value = input
		case 1:
			value = input + c
		case 2:
			value = input * (c + 2)
		case 3:
			value = input - c
		}
		return boolValue(applyCompare(gold, value, target))
	}, append(critical, c, target), nil
}

func buildBoolean(b *Builder, v int) (map[string]string, func(int64) Value, []int64, error) {
	x := b.input()
	a, d, e := int64(v%7-3), int64(v%5+1), int64(v%9-4)
	if v%3 == 1 {
		e = a + 2
	}
	boolA, boolB := cycle(boolOps, v+1), cycle(boolOps, v+2)
	left := b.binary("less_than", x, b.integer(a))
	right := b.binary("less_equal", x, b.integer(a))
	first := b.hole("h1", boolText(boolA, v), left, right, boolOps, nextAllowed(boolOps, boolA, v+2))
	ops := map[string]string{"h1": boolA}
	critical := []int64{a, d, e}
	if v%3 == 0 {
		b.root(b.ret(first))
		return ops, func(input int64) Value {
			lv := input < a
			rv := input <= a
			return boolValue(applyBool(boolA, lv, rv))
		}, critical, nil
	}
	var third int
	var thirdPredicate func(int64) bool
	if v%3 == 1 {
		third = b.binary("equal", x, b.integer(a))
		thirdPredicate = func(input int64) bool { return input == a }
	} else {
		third = b.binary("equal", x, b.integer(e))
		thirdPredicate = func(input int64) bool { return input == e }
	}
	firstLocal := b.let("combined_test", first)
	combinedLocal := b.local("combined_test")
	combined := b.hole("h2", boolText(boolB, v+1), combinedLocal, third, boolOps, nextAllowed(boolOps, boolB, v))
	ops["h2"] = boolB
	b.root(firstLocal, b.ret(combined))
	return ops, func(input int64) Value {
		lv := input < a
		rv := input <= a
		inner := applyBool(boolA, lv, rv)
		return boolValue(applyBool(boolB, inner, thirdPredicate(input)))
	}, critical, nil
}

func buildReassignment(b *Builder, v int) (map[string]string, func(int64) Value, []int64, error) {
	x := b.input()
	c, d := int64(v%7+1), int64(v%5+2)
	k := d + int64(v%3+1)
	goldA, goldB := cycle(intOps, v+1), cycle(intOps, v+2)
	fallbackA, fallbackB := nextAllowed(intOps, goldA, v), nextAllowed(intOps, goldB, v+1)
	initial := b.binary("add", x, b.integer(c))
	b.let("total", initial)
	old := b.local("total")
	updated := b.hole("h1", textFor(goldA, false, v+3), old, b.integer(d), intOps, fallbackA)
	assignIndex := b.assign("total", updated)
	ops := map[string]string{"h1": goldA}
	if v%4 == 0 {
		final := b.binary("multiply", b.local("total"), b.integer(k))
		b.root(0, assignIndex, b.ret(final))
		return ops, func(input int64) Value { return intValue(applyInt(goldA, input+c, d) * k) }, []int64{c, d}, nil
	}
	if v%4 == 1 {
		again := b.local("total")
		second := b.hole("h2", textFor(goldB, true, v+2), again, b.integer(k), intOps, fallbackB)
		secondAssign := b.assign("total", second)
		b.root(0, assignIndex, secondAssign, b.ret(b.local("total")))
		ops["h2"] = goldB
		return ops, func(input int64) Value { return intValue(applyInt(goldB, applyInt(goldA, input+c, d), k)) }, []int64{c, d, k}, nil
	}
	if v%4 == 2 {
		mid := b.binary("subtract", b.local("total"), b.integer(k))
		second := b.hole("h2", textFor(goldB, true, v+2), mid, b.integer(c), intOps, fallbackB)
		secondAssign := b.assign("total", second)
		b.root(0, assignIndex, secondAssign, b.ret(b.local("total")))
		ops["h2"] = goldB
		return ops, func(input int64) Value { return intValue(applyInt(goldB, applyInt(goldA, input+c, d)-k, c)) }, []int64{c, d, k}, nil
	}
	flag := b.binary("less_equal", b.local("total"), b.integer(int64(0)))
	positive := b.binary("add", b.local("total"), b.integer(k))
	negative := b.binary("subtract", b.local("total"), b.integer(k))
	thenStmt, elseStmt := b.ret(positive), b.ret(negative)
	conditional := b.branch(flag, []int{thenStmt}, []int{elseStmt})
	b.root(0, assignIndex, conditional)
	return ops, func(input int64) Value {
		value := applyInt(goldA, input+c, d)
		if value <= 0 {
			return intValue(value + k)
		}
		return intValue(value - k)
	}, []int64{c, d, -k, 0}, nil
}

func buildPiecewise(b *Builder, v int) (map[string]string, func(int64) Value, []int64, error) {
	x := b.input()
	t := thresholdInputs[(v*5+3)%len(thresholdInputs)]
	c := int64(v%7 + 1)
	if v%2 == 0 {
		c = -c
	}
	cmp := cycle(cmpOps, v+1)
	intOp := cycle(intOps, v+2)
	condition := b.hole("h1", compareText(cmp, v+4), x, b.integer(t), cmpOps, nextAllowed(cmpOps, cmp, v))
	thenValue := b.hole("h2", textFor(intOp, false, v+2), x, b.integer(c), intOps, nextAllowed(intOps, intOp, v+1))
	var elseValue int
	if v%4 == 0 {
		elseValue = b.binary("multiply", x, b.integer(int64(v%3+2)))
	} else if v%4 == 1 {
		elseValue = b.binary("subtract", b.integer(c), x)
	} else if v%4 == 2 {
		elseValue = b.binary("add", x, b.integer(-c-1))
	} else {
		scaled := b.binary("multiply", x, b.integer(-2))
		elseValue = b.binary("add", scaled, b.integer(1))
	}
	thenStmt, elseStmt := b.ret(thenValue), b.ret(elseValue)
	conditionStmt := b.branch(condition, []int{thenStmt}, []int{elseStmt})
	b.root(conditionStmt)
	return map[string]string{"h1": cmp, "h2": intOp}, func(input int64) Value {
		if applyCompare(cmp, input, t) {
			return intValue(applyInt(intOp, input, c))
		}
		switch v % 4 {
		case 0:
			return intValue(input * int64(v%3+2))
		case 1:
			return intValue(c - input)
		case 2:
			return intValue(input + (-c - 1))
		default:
			return intValue(input*-2 + 1)
		}
	}, []int64{t, c, t - 1, t + 1}, nil
}

func buildNested(b *Builder, v int) (map[string]string, func(int64) Value, []int64, error) {
	x := b.input()
	outerThreshold := thresholdInputs[(v*7+11)%len(thresholdInputs)]
	outerAllowed := []string{"less_equal", "less_than"}
	outerOp, innerOp := cycle(outerAllowed, v), cycle(cmpOps, v+1)
	innerThreshold := outerThreshold
	if outerOp == "less_than" {
		innerThreshold--
	}
	outer := b.hole("h1", compareText(outerOp, v+8), x, b.integer(outerThreshold), outerAllowed, nextAllowed(outerAllowed, outerOp, v))
	inner := b.hole("h2", compareText(innerOp, v+9), x, b.integer(innerThreshold), cmpOps, nextAllowed(cmpOps, innerOp, v+1))
	low := b.binary("subtract", x, b.integer(int64(v%5+1)))
	middle := b.binary("multiply", x, b.integer(int64(v%3+2)))
	high := b.binary("add", x, b.integer(int64(v%7+3)))
	lowRet, midRet, highRet := b.ret(low), b.ret(middle), b.ret(high)
	nested := b.branch(inner, []int{highRet}, []int{midRet})
	outerBranch := b.branch(outer, []int{nested}, []int{lowRet})
	b.root(outerBranch)
	return map[string]string{"h1": outerOp, "h2": innerOp}, func(input int64) Value {
		if applyCompare(outerOp, input, outerThreshold) {
			if applyCompare(innerOp, input, innerThreshold) {
				return intValue(input + int64(v%7+3))
			}
			return intValue(input * int64(v%3+2))
		}
		return intValue(input - int64(v%5+1))
	}, []int64{outerThreshold, innerThreshold, outerThreshold - 1, outerThreshold + 1, innerThreshold - 1, innerThreshold + 1}, nil
}

func buildPolynomial(b *Builder, v int) (map[string]string, func(int64) Value, []int64, error) {
	x := b.input()
	c, d := int64(v%5+1), int64(v%7+1)
	if v%2 == 0 {
		d = -d
	}
	product := b.binary("multiply", x, x)
	firstOp, secondOp := cycle(intOps, v+1), cycle(intOps, v+2)
	fallbackA, fallbackB := nextAllowed(intOps, firstOp, v), nextAllowed(intOps, secondOp, v+1)
	ops := map[string]string{"h1": firstOp}
	if v%4 == 0 {
		first := b.hole("h1", textFor(firstOp, false, v+5), product, b.integer(c), intOps, fallbackA)
		result := b.binary("add", first, b.integer(d))
		b.root(b.ret(result))
		return ops, func(input int64) Value { return intValue(applyInt(firstOp, input*input, c) + d) }, []int64{c, d}, nil
	}
	if v%4 == 1 {
		first := b.hole("h1", textFor(firstOp, false, v+5), x, b.integer(c), intOps, fallbackA)
		result := b.binary("multiply", product, first)
		b.root(b.ret(result))
		return ops, func(input int64) Value { return intValue(input * input * applyInt(firstOp, input, c)) }, []int64{c}, nil
	}
	if v%4 == 2 {
		first := b.hole("h1", textFor(firstOp, false, v+5), product, b.integer(c), intOps, fallbackA)
		second := b.hole("h2", textFor(secondOp, true, v+6), first, x, intOps, fallbackB)
		result := b.binary("subtract", second, b.integer(d))
		ops["h2"] = secondOp
		b.root(b.ret(result))
		return ops, func(input int64) Value {
			return intValue(applyInt(secondOp, applyInt(firstOp, input*input, c), input) - d)
		}, []int64{c, d}, nil
	}
	first := b.hole("h1", textFor(firstOp, false, v+5), x, b.integer(c), intOps, fallbackA)
	second := b.hole("h2", textFor(secondOp, true, v+6), product, first, intOps, fallbackB)
	thirdOp := cycle(intOps, v+3)
	third := b.hole("h3", textFor(thirdOp, false, v+7), second, b.integer(d), intOps, nextAllowed(intOps, thirdOp, v+2))
	ops["h2"], ops["h3"] = secondOp, thirdOp
	b.root(b.ret(third))
	return ops, func(input int64) Value {
		return intValue(applyInt(thirdOp, applyInt(secondOp, input*input, applyInt(firstOp, input, c)), d))
	}, []int64{c, d}, nil
}

func buildBoundary(b *Builder, v int) (map[string]string, func(int64) Value, []int64, error) {
	x := b.input()
	gold := make([]string, 4)
	for i := range gold {
		gold[i] = cycle(intOps, v+i+2)
	}
	if v%2 != 0 {
		gold[0] = "add"
	}
	fall := make([]string, 4)
	for i := range fall {
		fall[i] = nextAllowed(intOps, gold[i], v+i)
	}
	extreme := int64(math.MaxInt64)
	if v%2 != 0 {
		extreme = math.MinInt64
	}
	constants := []int64{extreme, int64(2*v + 3), -int64(2*v + 5), int64(2*v + 7)}
	values := []int{x}
	ops := make(map[string]string)
	var outputs []string
	count := 2
	if v%4 == 0 {
		count = 4
	}
	if v%4 == 1 {
		count = 3
	}
	for i := 0; i < count; i++ {
		left := values[len(values)-1]
		constant := b.integer(constants[i])
		if (v+i)%2 == 0 {
			left, constant = constant, left
		}
		holeID := "h" + strconv.Itoa(i+1)
		text := textFor(gold[i], (v+i)%2 == 0, v+i+11)
		allowed := intOps
		fallback := fall[i]
		if i == 0 && v%2 != 0 {
			allowed = []string{"add", "multiply"}
		}
		if i == 0 && v%2 != 0 {
			fallback = "multiply"
		}
		values = append(values, b.hole(holeID, text, left, constant, allowed, fallback))
		ops[holeID] = gold[i]
		outputs = append(outputs, holeID)
	}
	finalExpr := values[len(values)-1]
	if v%4 == 2 {
		finalExpr = b.binary("subtract", finalExpr, b.integer(int64(v+1)))
	}
	b.root(b.ret(finalExpr))
	critical := []int64{math.MinInt64, math.MinInt64 + 1, math.MaxInt64 - 1, math.MaxInt64, -1, 0, 1, int64(v + 2)}
	return ops, func(input int64) Value {
		value := input
		for i := range outputs {
			constant := constants[i]
			if (v+i)%2 == 0 {
				value = applyInt(gold[i], constant, value)
			} else {
				value = applyInt(gold[i], value, constant)
			}
		}
		if v%4 == 2 {
			value -= int64(v + 1)
		}
		return intValue(value)
	}, critical, nil
}

func cycle(values []string, index int) string { return values[index%len(values)] }

func nextAllowed(values []string, gold string, index int) string {
	for offset := 0; offset < len(values); offset++ {
		candidate := values[(index+offset)%len(values)]
		if candidate != gold {
			return candidate
		}
	}
	return gold
}

func applyInt(operation string, left, right int64) int64 {
	switch operation {
	case "add":
		return left + right
	case "subtract":
		return left - right
	case "multiply":
		return left * right
	default:
		panic("non-integer operation in integer formula: " + operation)
	}
}

func applyCompare(operation string, left, right int64) bool {
	switch operation {
	case "less_than":
		return left < right
	case "less_equal":
		return left <= right
	case "equal":
		return left == right
	default:
		panic("non-comparison operation in Boolean formula: " + operation)
	}
}

func applyBool(operation string, left, right bool) bool {
	switch operation {
	case "and":
		return left && right
	case "or":
		return left || right
	case "equal":
		return left == right
	default:
		panic("non-Boolean operation in Boolean formula: " + operation)
	}
}

func intValue(value int64) Value { return Value{typ: "Int", i: value} }
func boolValue(value bool) Value { return Value{typ: "Bool", b: value} }

func expected(value Value) Expected {
	if value.typ == "Int" {
		return Expected{Type: "Int", Int: ptrI64(value.i)}
	}
	return Expected{Type: "Bool", Bool: ptrBool(value.b)}
}

func textFor(operation string, reverse bool, salt int) string {
	english := []string{
		"Add the left quantity to the right quantity; both contributions raise the total.",
		"Subtract the right quantity from the left quantity; preserve the displayed order.",
		"Scale the left quantity by the right quantity, preserving the product.",
	}
	korean := []string{
		"왼쪽 값에 오른쪽 값을 더해 두 기여분을 합산하세요.",
		"왼쪽 값에서 오른쪽 값을 빼서 표시된 순서를 유지하세요.",
		"왼쪽 값과 오른쪽 값의 곱으로 크기를 조정하세요.",
	}
	index := 0
	switch operation {
	case "add":
		index = 0
	case "subtract":
		index = 1
	case "multiply":
		index = 2
	}
	text := english[index]
	if salt%2 != 0 {
		text = korean[index]
	}
	if reverse {
		text += " Keep the displayed operand order."
	}
	return text
}

func compareText(operation string, salt int) string {
	phrases := map[string][2]string{
		"less_than":  {"Check whether the left measurement is strictly below the right boundary.", "왼쪽 측정값이 오른쪽 경계보다 엄격히 작은지 확인하세요."},
		"less_equal": {"Check whether the left measurement is at or below the right boundary.", "왼쪽 측정값이 오른쪽 경계 이하인지 확인하세요."},
		"equal":      {"Check whether the two measurements match exactly.", "두 측정값이 정확히 같은지 확인하세요."},
	}
	index := 0
	if salt%2 != 0 {
		index = 1
	}
	return phrases[operation][index]
}

func boolText(operation string, salt int) string {
	phrases := map[string][2]string{
		"and":   {"Both conditions must hold for the combined result to hold.", "결합된 결과가 참이려면 두 조건이 모두 참이어야 합니다."},
		"or":    {"The combined result holds when either condition holds.", "두 조건 중 하나라도 참이면 결합된 결과가 참입니다."},
		"equal": {"Report whether the two condition results agree.", "두 조건 결과가 서로 일치하는지 알려주세요."},
	}
	index := 0
	if salt%2 != 0 {
		index = 1
	}
	return phrases[operation][index]
}

func makeTrainInputs(critical []int64) []int64 {
	values := make([]int64, 0, len(coreTrainInputs)+len(critical)*3)
	seen := make(map[int64]bool)
	add := func(value int64) {
		if !seen[value] {
			seen[value] = true
			values = append(values, value)
		}
	}
	for _, value := range coreTrainInputs {
		add(value)
	}
	for _, value := range critical {
		add(value)
		if value != math.MinInt64 {
			add(value - 1)
		}
		if value != math.MaxInt64 {
			add(value + 1)
		}
	}
	return values
}

func makeHeldoutInputs(index int, train []int64, plan Plan) []int64 {
	blocked := make(map[int64]bool, len(train))
	for _, value := range train {
		blocked[value] = true
	}
	negative := make([]int64, 0, 24)
	positive := make([]int64, 0, 24)
	appendUnique := func(target *[]int64, value int64) {
		if !blocked[value] {
			blocked[value] = true
			*target = append(*target, value)
		}
	}
	appendUnique(&negative, math.MinInt64+2)
	appendUnique(&positive, math.MaxInt64-2)
	boundaries := branchBoundaries(plan)
	for _, offset := range []int64{-2, 2, -3, 3} {
		for _, boundary := range boundaries {
			if offset < 0 && boundary < math.MinInt64-offset {
				continue
			}
			if offset > 0 && boundary > math.MaxInt64-offset {
				continue
			}
			candidate := boundary + offset
			if candidate < 0 {
				appendUnique(&negative, candidate)
			} else {
				appendUnique(&positive, candidate)
			}
		}
	}
	for _, value := range []int64{math.MinInt64 + 3, math.MinInt64 + 19, -9000000000000000001, -9000000000000000000, -1000002, -65537, -4097, -997, -43, -42, -41, -13, -12, -11, -9, -6, -4} {
		appendUnique(&negative, value)
	}
	for _, value := range []int64{math.MaxInt64 - 3, math.MaxInt64 - 19, 9000000000000000000, 9000000000000000001, 1000002, 65537, 4097, 997, 43, 42, 41, 13, 12, 11, 9, 6, 4} {
		appendUnique(&positive, value)
	}
	base := int64(8000000000000000 + index*10007)
	appendUnique(&positive, base)
	appendUnique(&negative, -base)
	selected := make([]int64, 0, 10)
	for i := 0; i < 5; i++ {
		selected = append(selected, negative[i], positive[i])
	}
	return selected
}

func branchBoundaries(plan Plan) []int64 {
	locals := make(map[string]int)
	assigned := make(map[string]bool)
	for _, stmt := range plan.Statements {
		if stmt.Kind == "let" && stmt.Expr != nil {
			locals[stmt.Name] = *stmt.Expr
		}
		if stmt.Kind == "assign" {
			assigned[stmt.Name] = true
		}
	}
	for name := range assigned {
		delete(locals, name)
	}
	var boundaries []int64
	for _, stmt := range plan.Statements {
		if stmt.Kind != "if" || stmt.Expr == nil {
			continue
		}
		condition := plan.Expressions[*stmt.Expr]
		if condition.Kind != "hole" && condition.Kind != "binary" {
			continue
		}
		if condition.Left == nil || condition.Right == nil {
			continue
		}
		left, leftOK := affineInput(plan, *condition.Left, locals, 0)
		right, rightOK := affineInput(plan, *condition.Right, locals, 0)
		if !leftOK || !rightOK {
			continue
		}
		if left.a == 1 || left.a == -1 {
			if right.a == 0 {
				boundaries = append(boundaries, solveAffineBoundary(left, right.b))
			}
		} else if right.a == 1 || right.a == -1 {
			if left.a == 0 {
				boundaries = append(boundaries, solveAffineBoundary(right, left.b))
			}
		}
	}
	return boundaries
}

type affineValue struct{ a, b int64 }

func affineInput(plan Plan, index int, locals map[string]int, depth int) (affineValue, bool) {
	if depth > 16 || index < 0 || index >= len(plan.Expressions) {
		return affineValue{}, false
	}
	expr := plan.Expressions[index]
	switch expr.Kind {
	case "input":
		return affineValue{a: 1}, true
	case "int":
		if expr.Int != nil {
			return affineValue{b: *expr.Int}, true
		}
		return affineValue{}, false
	case "local":
		child, ok := locals[expr.Name]
		if !ok {
			return affineValue{}, false
		}
		return affineInput(plan, child, locals, depth+1)
	case "binary":
		if expr.Left == nil || expr.Right == nil {
			return affineValue{}, false
		}
		left, leftOK := affineInput(plan, *expr.Left, locals, depth+1)
		right, rightOK := affineInput(plan, *expr.Right, locals, depth+1)
		if !leftOK || !rightOK {
			return affineValue{}, false
		}
		switch expr.Operation {
		case "add":
			return affineValue{a: left.a + right.a, b: left.b + right.b}, true
		case "subtract":
			return affineValue{a: left.a - right.a, b: left.b - right.b}, true
		case "multiply":
			if left.a == 0 {
				return affineValue{a: right.a * left.b, b: right.b * left.b}, true
			}
			if right.a == 0 {
				return affineValue{a: left.a * right.b, b: left.b * right.b}, true
			}
		}
	}
	return affineValue{}, false
}

func solveAffineBoundary(value affineValue, target int64) int64 {
	if value.a == 1 {
		return target - value.b
	}
	return value.b - target
}

func validateRow(row Row) error {
	if row.Plan.Schema != planSchema || row.Plan.ID != row.ID {
		return errors.New("plan identity or schema mismatch")
	}
	if !goIdentifier(row.Plan.Name) {
		return errors.New("plan name must be a Go identifier")
	}
	if len(row.Plan.Expressions) > 128 || len(row.Plan.Statements) > 128 || len(row.Plan.Root) == 0 {
		return errors.New("plan bounds or root are invalid")
	}
	if len(row.TrainingCases) < 7 || len(row.HeldoutCases) < 10 {
		return errors.New("training or heldout coverage is too small")
	}
	train := make(map[int64]bool)
	for _, item := range row.TrainingCases {
		train[item.Input] = true
		if err := validExpected(item.Expected, row.Plan.ResultType); err != nil {
			return err
		}
	}
	for _, item := range row.HeldoutCases {
		if train[item.Input] {
			return fmt.Errorf("train and heldout inputs overlap at %d", item.Input)
		}
		if err := validExpected(item.Expected, row.Plan.ResultType); err != nil {
			return err
		}
	}
	negative, nonnegative := 0, 0
	for _, item := range row.HeldoutCases {
		if item.Input < 0 {
			negative++
		} else {
			nonnegative++
		}
	}
	if negative != 5 || nonnegative != 5 {
		return errors.New("heldout inputs must have five negative and five nonnegative values")
	}
	return validateGraph(row)
}

func validateOracleAgreement(row Row, oracle func(int64) Value) error {
	program, err := bodyplan.Compile(toBodyPlan(row.Plan), row.OracleOperations)
	if err != nil {
		return fmt.Errorf("native body-plan compile rejected gold operations: %w", err)
	}
	cases := append(append([]Case(nil), row.TrainingCases...), row.HeldoutCases...)
	for _, item := range cases {
		want := oracle(item.Input)
		if want.typ != row.Plan.ResultType || !expectedEqual(expected(want), item.Expected) {
			return fmt.Errorf("handwritten oracle case disagrees at input %d", item.Input)
		}
		got, err := evaluatePlan(row.Plan, item.Input, row.OracleOperations)
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("plan and handwritten oracle disagree at input %d: plan=%+v oracle=%+v", item.Input, got, want)
		}
		native, err := program.Evaluate(item.Input)
		if err != nil {
			return err
		}
		if native.Type != decision.ValueType(want.typ) || (want.typ == "Int" && native.Int != want.i) || (want.typ == "Bool" && native.Bool != want.b) {
			return fmt.Errorf("native body-plan evaluator disagrees with handwritten oracle at input %d", item.Input)
		}
	}
	return nil
}

func validExpected(value Expected, resultType string) error {
	if value.Type != resultType {
		return errors.New("expected result type mismatch")
	}
	if value.Type == "Int" && (value.Int == nil || value.Bool != nil) {
		return errors.New("integer expectation must have an explicit int field only")
	}
	if value.Type == "Bool" && (value.Bool == nil || value.Int != nil) {
		return errors.New("Boolean expectation must have an explicit bool field only")
	}
	return nil
}

func expectedEqual(left, right Expected) bool {
	if left.Type != right.Type {
		return false
	}
	if left.Type == "Int" {
		return left.Int != nil && right.Int != nil && *left.Int == *right.Int
	}
	return left.Bool != nil && right.Bool != nil && *left.Bool == *right.Bool
}

func toBodyPlan(plan Plan) bodyplan.Plan {
	expressions := make([]bodyplan.Expr, len(plan.Expressions))
	for index, source := range plan.Expressions {
		expr := bodyplan.Expr{
			Kind: source.Kind, Name: source.Name, Operation: source.Operation,
			HoleID: source.HoleID, Text: source.Text,
			Allowed: append([]string(nil), source.Allowed...), Fallback: source.Fallback,
		}
		if source.Int != nil {
			expr.Int = *source.Int
		}
		if source.Bool != nil {
			expr.Bool = *source.Bool
		}
		if source.Left != nil {
			expr.Left = *source.Left
		}
		if source.Right != nil {
			expr.Right = *source.Right
		}
		expressions[index] = expr
	}
	statements := make([]bodyplan.Stmt, len(plan.Statements))
	for index, source := range plan.Statements {
		stmt := bodyplan.Stmt{Kind: source.Kind, Name: source.Name, Then: append([]int(nil), source.Then...), Else: append([]int(nil), source.Else...)}
		if source.Expr != nil {
			stmt.Expr = *source.Expr
		}
		statements[index] = stmt
	}
	return bodyplan.Plan{
		Schema: plan.Schema, ID: plan.ID, Name: plan.Name,
		ResultType:  decision.ValueType(plan.ResultType),
		Expressions: expressions, Statements: statements, Root: append([]int(nil), plan.Root...),
	}
}

func validateGraph(row Row) error {
	plan := row.Plan
	if len(plan.Expressions) == 0 || len(plan.Expressions) > 128 || len(plan.Statements) > 128 {
		return errors.New("expression or statement bounds exceeded")
	}
	usedExpr := make([]bool, len(plan.Expressions))
	for index, expr := range plan.Expressions {
		if expr.Kind == "binary" || expr.Kind == "hole" {
			if expr.Left == nil || expr.Right == nil || *expr.Left < 0 || *expr.Right < 0 || *expr.Left >= index || *expr.Right >= index {
				return fmt.Errorf("expression %d has a non-topological child", index)
			}
		}
		if expr.Kind == "hole" {
			if len(expr.Allowed) < 2 || len(expr.Allowed) > 3 || !contains(expr.Allowed, expr.Fallback) || expr.Text == "" || len(expr.Text) > 512 || !utf8Text(expr.Text) {
				return fmt.Errorf("hole %d violates closed-choice or text bounds", index)
			}
			if row.OracleOperations[expr.HoleID] == expr.Fallback {
				return fmt.Errorf("hole %q fallback must differ from its gold operation", expr.HoleID)
			}
		}
	}
	stmtParents := make([]int, len(plan.Statements))
	for _, root := range plan.Root {
		if err := addStmtEdge(stmtParents, root); err != nil {
			return err
		}
	}
	for _, stmt := range plan.Statements {
		if stmt.Kind == "if" {
			for _, child := range stmt.Then {
				if err := addStmtEdge(stmtParents, child); err != nil {
					return err
				}
			}
			for _, child := range stmt.Else {
				if err := addStmtEdge(stmtParents, child); err != nil {
					return err
				}
			}
		}
	}
	for index, count := range stmtParents {
		if count != 1 {
			return fmt.Errorf("statement %d is reachable %d times", index, count)
		}
	}
	seen := make(map[string]bool)
	for id := range row.OracleOperations {
		seen[id] = true
	}
	if len(seen) == 0 || len(seen) > 16 {
		return errors.New("hole count outside bounds")
	}
	for _, stmt := range plan.Statements {
		if stmt.Expr == nil || *stmt.Expr < 0 || *stmt.Expr >= len(plan.Expressions) {
			return fmt.Errorf("statement %q lacks a valid expression", stmt.Kind)
		}
		markExpression(plan.Expressions, usedExpr, *stmt.Expr)
		if stmt.Kind == "if" && (len(stmt.Then) == 0 || len(stmt.Else) == 0) {
			return errors.New("if statements require both branches")
		}
	}
	for index, used := range usedExpr {
		if !used {
			return fmt.Errorf("expression %d is unused", index)
		}
	}
	declared, referenced := make(map[string]bool), make(map[string]bool)
	for _, stmt := range plan.Statements {
		if stmt.Kind == "let" {
			declared[stmt.Name] = true
		}
		if stmt.Kind == "assign" && !declared[stmt.Name] {
			return fmt.Errorf("assignment to undeclared local %q", stmt.Name)
		}
	}
	for _, expr := range plan.Expressions {
		if expr.Kind == "local" {
			referenced[expr.Name] = true
		}
	}
	for name := range declared {
		if !referenced[name] {
			return fmt.Errorf("local %q is never read", name)
		}
	}
	for name := range referenced {
		if !declared[name] {
			return fmt.Errorf("local reference %q has no declaration", name)
		}
	}
	for id := range seen {
		found := false
		for _, expr := range plan.Expressions {
			if expr.Kind == "hole" && expr.HoleID == id {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("oracle operation references unknown hole %q", id)
		}
	}
	return nil
}

func addStmtEdge(parents []int, index int) error {
	if index < 0 || index >= len(parents) {
		return errors.New("statement edge out of range")
	}
	parents[index]++
	return nil
}

func markExpression(expressions []Expression, used []bool, index int) {
	if used[index] {
		return
	}
	used[index] = true
	expr := expressions[index]
	if expr.Left != nil {
		markExpression(expressions, used, *expr.Left)
	}
	if expr.Right != nil {
		markExpression(expressions, used, *expr.Right)
	}
}

func contains(values []string, item string) bool {
	for _, value := range values {
		if value == item {
			return true
		}
	}
	return false
}

func utf8Text(text string) bool { return strings.ToValidUTF8(text, "�") == text }

func goIdentifier(value string) bool {
	if value == "" || !((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z') || value[0] == '_') {
		return false
	}
	for i := 1; i < len(value); i++ {
		b := value[i]
		if !((b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_') {
			return false
		}
	}
	return true
}

func validateGoldDiscrimination(row Row) error {
	holes := make([]Expression, 0)
	for _, expr := range row.Plan.Expressions {
		if expr.Kind == "hole" {
			holes = append(holes, expr)
		}
	}
	if len(holes) != len(row.OracleOperations) {
		return errors.New("hole and oracle maps differ")
	}
	assignments := make([]map[string]string, 0, 128)
	var enumerate func(int, map[string]string)
	enumerate = func(index int, current map[string]string) {
		if index == len(holes) {
			copy := make(map[string]string, len(current))
			for key, value := range current {
				copy[key] = value
			}
			assignments = append(assignments, copy)
			return
		}
		hole := holes[index]
		for _, option := range hole.Allowed {
			current[hole.HoleID] = option
			enumerate(index+1, current)
		}
	}
	enumerate(0, make(map[string]string))
	gold := row.OracleOperations
	goldOutputs := make([]Value, len(row.TrainingCases))
	for i, item := range row.TrainingCases {
		value, err := evaluatePlan(row.Plan, item.Input, gold)
		if err != nil {
			return err
		}
		goldOutputs[i] = value
	}
	matched := 0
	for _, candidate := range assignments {
		if _, err := bodyplan.Compile(toBodyPlan(row.Plan), candidate); err != nil {
			return fmt.Errorf("native body-plan compile rejected mapping %v: %w", candidate, err)
		}
		isGold := true
		for id, operation := range gold {
			if candidate[id] != operation {
				isGold = false
				break
			}
		}
		if isGold {
			matched++
			continue
		}
		for i, item := range row.TrainingCases {
			value, err := evaluatePlan(row.Plan, item.Input, candidate)
			if err != nil {
				return err
			}
			if value != goldOutputs[i] {
				break
			}
			if i == len(row.TrainingCases)-1 {
				return fmt.Errorf("training cases do not distinguish candidate mapping %v", candidate)
			}
		}
	}
	if matched != 1 {
		return errors.New("gold operation assignment is not unique")
	}
	return nil
}

func evaluatePlan(plan Plan, input int64, choices map[string]string) (Value, error) {
	env := make(map[string]Value)
	returned, result := false, Value{}
	var evalExpr func(int) (Value, error)
	evalExpr = func(index int) (Value, error) {
		expr := plan.Expressions[index]
		switch expr.Kind {
		case "input":
			return intValue(input), nil
		case "int":
			return intValue(*expr.Int), nil
		case "bool":
			return boolValue(*expr.Bool), nil
		case "local":
			value, ok := env[expr.Name]
			if !ok {
				return Value{}, fmt.Errorf("local %q is out of scope", expr.Name)
			}
			return value, nil
		case "binary", "hole":
			left, err := evalExpr(*expr.Left)
			if err != nil {
				return Value{}, err
			}
			right, err := evalExpr(*expr.Right)
			if err != nil {
				return Value{}, err
			}
			operation := expr.Operation
			if expr.Kind == "hole" {
				operation = choices[expr.HoleID]
			}
			if left.typ == "Int" && right.typ == "Int" {
				switch operation {
				case "add", "subtract", "multiply":
					return intValue(applyInt(operation, left.i, right.i)), nil
				case "less_than", "less_equal", "equal":
					return boolValue(applyCompare(operation, left.i, right.i)), nil
				}
			}
			if left.typ == "Bool" && right.typ == "Bool" {
				return boolValue(applyBool(operation, left.b, right.b)), nil
			}
			return Value{}, fmt.Errorf("type mismatch for operation %q", operation)
		default:
			return Value{}, fmt.Errorf("unknown expression kind %q", expr.Kind)
		}
	}
	var execute func([]int) error
	execute = func(indices []int) error {
		for _, index := range indices {
			stmt := plan.Statements[index]
			value, err := evalExpr(*stmt.Expr)
			if err != nil {
				return err
			}
			switch stmt.Kind {
			case "let", "assign":
				env[stmt.Name] = value
			case "return":
				returned, result = true, value
				return nil
			case "if":
				branch := stmt.Else
				if value.typ != "Bool" {
					return errors.New("if condition is not Bool")
				}
				if value.b {
					branch = stmt.Then
				}
				if err := execute(branch); err != nil {
					return err
				}
				if returned {
					return nil
				}
			default:
				return fmt.Errorf("unknown statement kind %q", stmt.Kind)
			}
		}
		return nil
	}
	if err := execute(plan.Root); err != nil {
		return Value{}, err
	}
	if !returned {
		return Value{}, errors.New("plan did not return")
	}
	if result.typ != plan.ResultType {
		return Value{}, errors.New("result type mismatch")
	}
	return result, nil
}

func canonicalIntent(row Row) string {
	plan := row.Plan
	plan.Expressions = append([]Expression(nil), plan.Expressions...)
	plan.Statements = append([]Statement(nil), plan.Statements...)
	localNames := make(map[string]string)
	for _, statement := range plan.Statements {
		if statement.Kind == "let" {
			localNames[statement.Name] = fmt.Sprintf("local-%d", len(localNames)+1)
		}
	}
	for i := range plan.Statements {
		if alias, ok := localNames[plan.Statements[i].Name]; ok {
			plan.Statements[i].Name = alias
		} else {
			plan.Statements[i].Name = ""
		}
	}
	holeNames := make(map[string]string)
	for i := range plan.Expressions {
		if plan.Expressions[i].Kind == "hole" {
			holeNames[plan.Expressions[i].HoleID] = fmt.Sprintf("hole-%d", len(holeNames)+1)
		}
	}
	for i := range plan.Expressions {
		if plan.Expressions[i].Kind == "input" {
			plan.Expressions[i].Name = "input"
		} else if plan.Expressions[i].Kind == "local" {
			plan.Expressions[i].Name = localNames[plan.Expressions[i].Name]
		} else {
			plan.Expressions[i].Name = ""
		}
		if plan.Expressions[i].Kind == "hole" {
			plan.Expressions[i].HoleID = holeNames[plan.Expressions[i].HoleID]
		}
		plan.Expressions[i].Text = ""
		plan.Expressions[i].Fallback = ""
	}
	plan.ID, plan.Name = "", ""
	operations := make(map[string]string, len(row.OracleOperations))
	for id, operation := range row.OracleOperations {
		operations[holeNames[id]] = operation
	}
	compact, _ := json.Marshal(struct {
		Plan Plan              `json:"plan"`
		Ops  map[string]string `json:"ops"`
	}{Plan: plan, Ops: operations})
	return string(compact)
}

func encodeRows(rows []Row) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

func makeManifest(rows []Row, jsonl []byte) (map[string]any, error) {
	sourcePath, err := sourcePath()
	if err != nil {
		return nil, err
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	families := make(map[string]int)
	totalTraining, totalHeldout, maxExpressions, maxStatements, maxHoles := 0, 0, 0, 0, 0
	searchSpaces := make(map[string]int)
	largeSpaces, totalCandidateMappings := 0, 0
	caseSigns := map[string]map[string]int{
		"training": {"negative": 0, "zero": 0, "positive": 0},
		"heldout":  {"negative": 0, "zero": 0, "positive": 0},
	}
	for _, row := range rows {
		families[row.Family]++
		totalTraining += len(row.TrainingCases)
		totalHeldout += len(row.HeldoutCases)
		for _, item := range row.TrainingCases {
			caseSigns["training"][sign(item.Input)]++
		}
		for _, item := range row.HeldoutCases {
			caseSigns["heldout"][sign(item.Input)]++
		}
		if len(row.Plan.Expressions) > maxExpressions {
			maxExpressions = len(row.Plan.Expressions)
		}
		if len(row.Plan.Statements) > maxStatements {
			maxStatements = len(row.Plan.Statements)
		}
		holes := 0
		combinations := 1
		for _, expr := range row.Plan.Expressions {
			if expr.Kind == "hole" {
				holes++
				combinations *= len(expr.Allowed)
			}
		}
		if holes > maxHoles {
			maxHoles = holes
		}
		searchSpaces[strconv.Itoa(combinations)]++
		totalCandidateMappings += combinations
		if combinations > 64 {
			largeSpaces++
		}
	}
	return map[string]any{
		"schema":           "gooo/typed-body-plan-cohort-manifest/v1",
		"plan_schema":      planSchema,
		"cohort_sha256":    hash(jsonl),
		"generator_sha256": hash(source),
		"plan_count":       len(rows), "distinct_canonical_intents": len(rows),
		"families":         families,
		"train_case_count": totalTraining, "heldout_case_count": totalHeldout,
		"training_inputs_global":            coreTrainInputs,
		"result_types":                      map[string]int{"Int": countResult(rows, "Int"), "Bool": countResult(rows, "Bool")},
		"maxima":                            map[string]int{"expressions": maxExpressions, "statements": maxStatements, "holes": maxHoles},
		"cartesian_search_space_sizes":      searchSpaces,
		"plans_over_64_candidate_mappings":  largeSpaces,
		"native_preflight":                  map[string]int{"plans": len(rows), "candidate_mappings_compiled": totalCandidateMappings, "gold_evaluation_cases": totalTraining + totalHeldout},
		"case_sign_counts":                  caseSigns,
		"uniqueness_definition":             "Canonical intent preserves result type, typed expression graph and operand order, literals, fixed operations, hole locations/options, statement/control-flow topology, and gold operation assignment. It alpha-normalizes local and hole names, removes plan IDs/names, natural-language wording, and fallback choice; renaming, paraphrasing, or fallback changes do not create new intent.",
		"fallback_policy":                   "Fallback is a deterministic valid allowed operation chosen by rotating the declared option list from an index-based offset and skipping the gold operation. It deliberately differs from gold to make the unassisted fallback baseline informative; it is not a model prediction and is excluded from inference input.",
		"training_and_heldout_policy":       "Every row has disjoint inputs. Training includes a fixed signed-boundary core plus scenario critical values and is the only set used for candidate-discrimination checks. Each plan has five negative and five positive heldout inputs. Where an if condition has a solvable affine boundary, the heldout candidate pool includes inputs two and three units around that boundary when disjoint from training; the pool also includes MinInt64+2/+3 and MaxInt64-2/-3 probes. Heldout inputs never participate in candidate selection.",
		"superseded_initial_draft":          map[string]any{"status": "PRE_INFERENCE_DRAFT_NOT_SCORED", "cohort_sha256": "650130fbca5b1731370b2b0aeb2d00267e0545db11452a2ae67a4b6e368d1730", "manifest_sha256": "c454f96bece12e069fee5df0d749c55949a85492820092bdbb1fefae2e1b7ae1", "generator_sha256": "618eb76063be986c47c9d766d3c9dc158666c2919807f9c0893fd03f22799847", "reason": "Canonical intent normalization mutated the shared expression slice and erased hole Text/Fallback; the heldout pool was negative-first. Preserved in superseded-initial-cohort/."},
		"earlier_intermediate_not_retained": map[string]string{"status": "OBSERVED_HASHES_ONLY_NOT_RETAINED", "cohort_sha256": "56b43b9845ad45dcbdaa230e797228b54d998810ed9c5a35d2d198b07b141956", "generator_sha256": "1273ccedbbbd84d6fb3ecfbc2921c73dc1cad4b2508ad59e6851f690c153ac22"},
		"reference_limitations":             "Expected cases come from handwritten Go mathematical formulas in generate.go. The reference is independent of the body-plan evaluator and Gooo emitter, but it is correlated with the plan designer's intended mathematics; it is not an independent language implementation or a proof of full-domain correctness.",
		"model_input_boundary":              "For each hole, inference receives only its natural-language Text and the descriptions of its Allowed operations, within the plan's required type context. Do not provide IDs, family, oracle_operations, fallback, expected values, training cases, heldout cases, or manifest fields to the model. Training cases may be used only by the bounded search arm after the model proposes initial choices.",
		"supported_scope":                   "Finite one-input int64 expression holes with a closed operation set; this is not arbitrary code generation or full int64-domain proof.",
	}, nil
}

func countResult(rows []Row, result string) int {
	count := 0
	for _, row := range rows {
		if row.Plan.ResultType == result {
			count++
		}
	}
	return count
}

func sign(value int64) string {
	if value < 0 {
		return "negative"
	}
	if value == 0 {
		return "zero"
	}
	return "positive"
}

func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
