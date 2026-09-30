// Package bodyplan validates, renders, and interprets small typed expression
// bodies. Plans contain no source fragments: identifiers, operations, and
// control flow all come from closed typed fields.
package bodyplan

import (
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const (
	Schema      = "gooo/typed-body-plan/v1"
	maxExprs    = 128
	maxStmts    = 128
	maxHoles    = 16
	maxDepth    = 16
	maxHoleText = 512
)

const (
	ExprInput  = "input"
	ExprInt    = "int"
	ExprBool   = "bool"
	ExprLocal  = "local"
	ExprBinary = "binary"
	ExprHole   = "hole"

	StmtLet    = "let"
	StmtAssign = "assign"
	StmtIf     = "if"
	StmtReturn = "return"
)

var operations = map[string]string{
	"add":        "+",
	"subtract":   "-",
	"multiply":   "*",
	"less_than":  "<",
	"less_equal": "<=",
	"equal":      "==",
	"and":        "&&",
	"or":         "||",
}

// Plan is a bounded arena of typed expressions and statements. Expression
// children refer to earlier Expressions entries; statement control flow uses
// indices into Statements.
type Plan struct {
	Schema      string             `json:"schema"`
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	ResultType  decision.ValueType `json:"result_type"`
	Expressions []Expr             `json:"expressions"`
	Statements  []Stmt             `json:"statements"`
	Root        []int              `json:"root"`
}

// Expr is one typed expression node. Hole nodes are binary operations whose
// operation is selected from Allowed by Compile.
type Expr struct {
	Kind      string   `json:"kind"`
	Name      string   `json:"name,omitempty"`
	Int       int64    `json:"int,omitempty"`
	Bool      bool     `json:"bool,omitempty"`
	Operation string   `json:"operation,omitempty"`
	Left      int      `json:"left,omitempty"`
	Right     int      `json:"right,omitempty"`
	HoleID    string   `json:"hole_id,omitempty"`
	Text      string   `json:"text,omitempty"`
	Allowed   []string `json:"allowed,omitempty"`
	Fallback  string   `json:"fallback,omitempty"`
}

// Stmt is a statement node. Then and Else contain statement arena indices and
// are meaningful only for if statements.
type Stmt struct {
	Kind string `json:"kind"`
	Name string `json:"name,omitempty"`
	Expr int    `json:"expr,omitempty"`
	Then []int  `json:"then,omitempty"`
	Else []int  `json:"else,omitempty"`
}

// HoleSpec is the validated model-selection boundary for one operator hole.
// ResultType is shared by every Allowed operation for these operand types.
type HoleSpec struct {
	ID         string             `json:"id"`
	Text       string             `json:"text"`
	LeftType   decision.ValueType `json:"left_type"`
	RightType  decision.ValueType `json:"right_type"`
	ResultType decision.ValueType `json:"result_type"`
	Allowed    []string           `json:"allowed"`
	Fallback   string             `json:"fallback"`
}

// Value is the interpreter's closed runtime value representation.
type Value struct {
	Type decision.ValueType `json:"type"`
	Int  int64              `json:"int,omitempty"`
	Bool bool               `json:"bool,omitempty"`
}

// Program is an immutable compiled plan. Its internal slices and maps are
// private copies, so concurrent Evaluate calls do not share mutable state.
type Program struct {
	plan       Plan
	choices    map[string]string
	goSource   string
	goooSource string
}

// Holes validates the plan and returns unique typed operator holes in
// expression-arena order.
func (plan Plan) Holes() ([]HoleSpec, error) {
	validated, err := analyze(plan)
	if err != nil {
		return nil, err
	}
	holes := make([]HoleSpec, 0, len(validated.holes))
	for _, index := range validated.holeOrder {
		hole := validated.holes[plan.Expressions[index].HoleID]
		hole.Allowed = append([]string(nil), hole.Allowed...)
		holes = append(holes, hole)
	}
	return holes, nil
}

// Compile validates the entire plan, requires one allowed choice for every
// hole, and creates deterministic Go and Gooo source.
func Compile(plan Plan, choices map[string]string) (*Program, error) {
	validated, err := analyze(plan)
	if err != nil {
		return nil, err
	}
	if len(choices) != len(validated.holes) {
		return nil, errors.New("choices must name every hole exactly once")
	}
	ownedChoices := make(map[string]string, len(choices))
	for id, choice := range choices {
		hole, ok := validated.holes[id]
		if !ok {
			return nil, fmt.Errorf("choice names unknown hole %q", id)
		}
		if !contains(hole.Allowed, choice) {
			return nil, fmt.Errorf("choice %q is not allowed for hole %q", choice, id)
		}
		ownedChoices[id] = choice
	}
	for id := range validated.holes {
		if _, ok := ownedChoices[id]; !ok {
			return nil, fmt.Errorf("choice for hole %q is missing", id)
		}
	}

	ownedPlan := clonePlan(plan)
	goBody, err := renderSequence(ownedPlan, ownedChoices, ownedPlan.Root, renderGo, 0)
	if err != nil {
		return nil, err
	}
	goSource, err := format.Source([]byte("package generated\n\nfunc " + ownedPlan.Name + "(input int64) " + goType(ownedPlan.ResultType) + " {\n" + goBody + "}\n"))
	if err != nil {
		return nil, fmt.Errorf("generated Go source is invalid: %w", err)
	}
	if err := checkGoSource(goSource); err != nil {
		return nil, fmt.Errorf("generated Go source failed type checking: %w", err)
	}
	goooBody, err := renderSequence(ownedPlan, ownedChoices, ownedPlan.Root, renderGooo, -1)
	if err != nil {
		return nil, err
	}
	resultEntity := "Integer"
	if ownedPlan.ResultType == decision.TypeBool {
		resultEntity = "Boolean"
	}
	goooSource := "package bodyplan\nnamespace bodyplan\n\n" +
		"entity Integer id \"bodyplan://entity/integer\"\n" +
		"entity Boolean id \"bodyplan://entity/boolean\"\n\n" +
		"activity " + ownedPlan.Name + "(Integer) -> " + resultEntity + " computes " + strconv.Quote(strings.TrimSuffix(goooBody, "\n")) + "\n"
	return &Program{plan: ownedPlan, choices: ownedChoices, goSource: string(goSource), goooSource: goooSource}, nil
}

// GoSource returns package generated source for the compiled body.
func (program Program) GoSource() string { return program.goSource }

// GoooSource returns the package bodyplan declaration and JSON-quoted body.
func (program Program) GoooSource() string { return program.goooSource }

// Evaluate executes the immutable plan with Go int64 overflow and Boolean
// short-circuit semantics.
func (program Program) Evaluate(input int64) (Value, error) {
	if program.goSource == "" || program.goooSource == "" {
		return Value{}, errors.New("program is not compiled")
	}
	environment := &runtimeScope{values: map[string]Value{"input": {Type: decision.TypeInt, Int: input}}}
	value, returned, err := executeSequence(program.plan, program.choices, program.plan.Root, environment)
	if err != nil {
		return Value{}, err
	}
	if !returned || value.Type != program.plan.ResultType {
		return Value{}, errors.New("compiled body did not return its declared result type")
	}
	return value, nil
}

type validatedPlan struct {
	holes     map[string]HoleSpec
	holeOrder []int
	usedExpr  []bool
	seenStmt  []bool
}

type typeScope struct {
	parent *typeScope
	values map[string]decision.ValueType
}

func analyze(plan Plan) (*validatedPlan, error) {
	if plan.Schema != Schema {
		return nil, fmt.Errorf("plan schema must be %q", Schema)
	}
	if plan.ID == "" || !utf8.ValidString(plan.ID) || len(plan.ID) > 128 || hasControl(plan.ID) {
		return nil, errors.New("plan ID must be printable UTF-8 of at most 128 bytes")
	}
	if !validIdentifier(plan.Name) {
		return nil, errors.New("activity name must be a valid Go identifier")
	}
	if plan.ResultType != decision.TypeInt && plan.ResultType != decision.TypeBool {
		return nil, errors.New("result type must be Int or Bool")
	}
	if len(plan.Expressions) == 0 || len(plan.Expressions) > maxExprs {
		return nil, fmt.Errorf("expression count must be between 1 and %d", maxExprs)
	}
	if len(plan.Statements) == 0 || len(plan.Statements) > maxStmts {
		return nil, fmt.Errorf("statement count must be between 1 and %d", maxStmts)
	}
	if len(plan.Root) == 0 || len(plan.Root) > maxStmts {
		return nil, errors.New("root must contain at least one statement")
	}

	validated := &validatedPlan{
		holes:    make(map[string]HoleSpec),
		usedExpr: make([]bool, len(plan.Expressions)),
		seenStmt: make([]bool, len(plan.Statements)),
	}
	inputCount := 0
	seenHoleIDs := make(map[string]bool)
	for index, expression := range plan.Expressions {
		if err := validateExprShape(expression, index); err != nil {
			return nil, fmt.Errorf("expression %d: %w", index, err)
		}
		if expression.Kind == ExprInput {
			inputCount++
		}
		if expression.Kind == ExprHole {
			if !validHoleID(expression.HoleID) {
				return nil, fmt.Errorf("expression %d has invalid hole ID", index)
			}
			if seenHoleIDs[expression.HoleID] {
				return nil, fmt.Errorf("hole ID %q appears in more than one expression", expression.HoleID)
			}
			seenHoleIDs[expression.HoleID] = true
			if len(validated.holeOrder) >= maxHoles {
				return nil, fmt.Errorf("hole count exceeds %d", maxHoles)
			}
			validated.holeOrder = append(validated.holeOrder, index)
		}
	}
	if inputCount != 1 {
		return nil, errors.New("plan must contain exactly one Integer input named input")
	}
	for index, statement := range plan.Statements {
		if err := validateStatementShape(statement, index); err != nil {
			return nil, err
		}
	}
	if err := validateStatementTree(plan, validated, plan.Root, 0); err != nil {
		return nil, err
	}
	for index, seen := range validated.seenStmt {
		if !seen {
			return nil, fmt.Errorf("statement %d is unused or unreachable", index)
		}
	}
	rootScope := &typeScope{values: map[string]decision.ValueType{"input": decision.TypeInt}}
	terminates, err := validateSequence(plan, validated, plan.Root, rootScope, 0)
	if err != nil {
		return nil, err
	}
	if !terminates {
		return nil, errors.New("every body path must return a value")
	}
	for index, used := range validated.usedExpr {
		if !used {
			return nil, fmt.Errorf("expression %d is unused", index)
		}
	}
	return validated, nil
}

func validateStatementTree(plan Plan, validated *validatedPlan, sequence []int, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("statement nesting exceeds %d", maxDepth)
	}
	for _, index := range sequence {
		if index < 0 || index >= len(plan.Statements) {
			return fmt.Errorf("statement index %d is out of range", index)
		}
		if validated.seenStmt[index] {
			return fmt.Errorf("statement %d is referenced more than once", index)
		}
		validated.seenStmt[index] = true
		statement := plan.Statements[index]
		if statement.Kind == StmtIf {
			if err := validateStatementTree(plan, validated, statement.Then, depth+1); err != nil {
				return err
			}
			if err := validateStatementTree(plan, validated, statement.Else, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSequence(plan Plan, validated *validatedPlan, sequence []int, scope *typeScope, depth int) (bool, error) {
	if depth > maxDepth {
		return false, fmt.Errorf("statement nesting exceeds %d", maxDepth)
	}
	for position, index := range sequence {
		statement := plan.Statements[index]
		terminates := false
		switch statement.Kind {
		case StmtLet:
			if !validLocalName(statement.Name) {
				return false, fmt.Errorf("statement %d has invalid let name", index)
			}
			if _, found := scope.lookup(statement.Name); found {
				return false, fmt.Errorf("let %q shadows a visible name", statement.Name)
			}
			expressionType, err := typeExpression(plan, validated, statement.Expr, scope, depth)
			if err != nil {
				return false, fmt.Errorf("statement %d: %w", index, err)
			}
			scope.values[statement.Name] = expressionType
		case StmtAssign:
			if !validLocalName(statement.Name) || statement.Name == "input" {
				return false, fmt.Errorf("statement %d has invalid assignment target", index)
			}
			currentType, found := scope.lookup(statement.Name)
			if !found {
				return false, fmt.Errorf("assignment target %q is not in scope", statement.Name)
			}
			expressionType, err := typeExpression(plan, validated, statement.Expr, scope, depth)
			if err != nil {
				return false, fmt.Errorf("statement %d: %w", index, err)
			}
			if currentType != expressionType {
				return false, fmt.Errorf("assignment to %q changes its type", statement.Name)
			}
		case StmtIf:
			conditionType, err := typeExpression(plan, validated, statement.Expr, scope, depth)
			if err != nil {
				return false, fmt.Errorf("statement %d: %w", index, err)
			}
			if conditionType != decision.TypeBool {
				return false, fmt.Errorf("if condition %d must have type Bool", index)
			}
			thenScope := &typeScope{parent: scope, values: make(map[string]decision.ValueType)}
			elseScope := &typeScope{parent: scope, values: make(map[string]decision.ValueType)}
			thenTerminates, err := validateSequence(plan, validated, statement.Then, thenScope, depth+1)
			if err != nil {
				return false, err
			}
			elseTerminates, err := validateSequence(plan, validated, statement.Else, elseScope, depth+1)
			if err != nil {
				return false, err
			}
			terminates = thenTerminates && elseTerminates
		case StmtReturn:
			resultType, err := typeExpression(plan, validated, statement.Expr, scope, depth)
			if err != nil {
				return false, fmt.Errorf("statement %d: %w", index, err)
			}
			if resultType != plan.ResultType {
				return false, fmt.Errorf("return statement %d has type %s, want %s", index, resultType, plan.ResultType)
			}
			terminates = true
		default:
			return false, fmt.Errorf("statement %d has unknown kind %q", index, statement.Kind)
		}
		if terminates && position != len(sequence)-1 {
			return false, fmt.Errorf("statement %d makes later statements unreachable", index)
		}
		if terminates {
			return true, nil
		}
	}
	return false, nil
}

func typeExpression(plan Plan, validated *validatedPlan, index int, scope *typeScope, depth int) (decision.ValueType, error) {
	if depth > maxDepth {
		return "", fmt.Errorf("expression nesting exceeds %d", maxDepth)
	}
	if index < 0 || index >= len(plan.Expressions) {
		return "", fmt.Errorf("expression index %d is out of range", index)
	}
	validated.usedExpr[index] = true
	expression := plan.Expressions[index]
	switch expression.Kind {
	case ExprInput:
		return decision.TypeInt, nil
	case ExprInt:
		return decision.TypeInt, nil
	case ExprBool:
		return decision.TypeBool, nil
	case ExprLocal:
		valueType, found := scope.lookup(expression.Name)
		if !found {
			return "", fmt.Errorf("local %q is not in scope", expression.Name)
		}
		return valueType, nil
	case ExprBinary, ExprHole:
		if expression.Left >= index || expression.Right >= index || expression.Left < 0 || expression.Right < 0 {
			return "", errors.New("expression children must refer to earlier arena entries")
		}
		leftType, err := typeExpression(plan, validated, expression.Left, scope, depth+1)
		if err != nil {
			return "", err
		}
		rightType, err := typeExpression(plan, validated, expression.Right, scope, depth+1)
		if err != nil {
			return "", err
		}
		operation := expression.Operation
		if expression.Kind == ExprHole {
			operation = expression.Fallback
		}
		ir, err := decision.BuildTypedBinary(operation, decision.Identifier{Name: "lhs", Type: leftType}, decision.Identifier{Name: "rhs", Type: rightType})
		if err != nil {
			return "", fmt.Errorf("invalid %s operation: %w", expression.Kind, err)
		}
		if expression.Kind == ExprHole {
			var commonResult decision.ValueType
			for _, candidate := range expression.Allowed {
				candidateIR, candidateErr := decision.BuildTypedBinary(candidate, decision.Identifier{Name: "lhs", Type: leftType}, decision.Identifier{Name: "rhs", Type: rightType})
				if candidateErr != nil {
					return "", fmt.Errorf("hole %q candidate %q is ill typed: %w", expression.HoleID, candidate, candidateErr)
				}
				if commonResult == "" {
					commonResult = candidateIR.ResultType
				} else if commonResult != candidateIR.ResultType {
					return "", fmt.Errorf("hole %q candidates do not share one result type", expression.HoleID)
				}
			}
			if commonResult != ir.ResultType {
				return "", fmt.Errorf("hole %q fallback result type is inconsistent", expression.HoleID)
			}
			spec := HoleSpec{ID: expression.HoleID, Text: expression.Text, LeftType: leftType, RightType: rightType, ResultType: commonResult, Allowed: append([]string(nil), expression.Allowed...), Fallback: expression.Fallback}
			if prior, exists := validated.holes[expression.HoleID]; exists {
				if !sameHole(prior, spec) {
					return "", fmt.Errorf("hole %q has context-dependent types", expression.HoleID)
				}
			} else {
				validated.holes[expression.HoleID] = spec
			}
		}
		return ir.ResultType, nil
	default:
		return "", fmt.Errorf("unknown expression kind %q", expression.Kind)
	}
}

func validateExprShape(expression Expr, index int) error {
	priorChild := func(child int) bool { return child >= 0 && child < index }
	switch expression.Kind {
	case ExprInput:
		if expression.Name != "input" || expression.Int != 0 || expression.Bool || expression.Operation != "" || expression.Left != 0 || expression.Right != 0 || expression.HoleID != "" || expression.Text != "" || len(expression.Allowed) != 0 || expression.Fallback != "" {
			return errors.New("input must be the Integer named input with no extra fields")
		}
	case ExprInt:
		if expression.Name != "" || expression.Bool || expression.Operation != "" || expression.Left != 0 || expression.Right != 0 || expression.HoleID != "" || expression.Text != "" || len(expression.Allowed) != 0 || expression.Fallback != "" {
			return errors.New("integer literal has irrelevant fields")
		}
	case ExprBool:
		if expression.Name != "" || expression.Int != 0 || expression.Operation != "" || expression.Left != 0 || expression.Right != 0 || expression.HoleID != "" || expression.Text != "" || len(expression.Allowed) != 0 || expression.Fallback != "" {
			return errors.New("Boolean literal has irrelevant fields")
		}
	case ExprLocal:
		if !validLocalName(expression.Name) || expression.Int != 0 || expression.Bool || expression.Operation != "" || expression.Left != 0 || expression.Right != 0 || expression.HoleID != "" || expression.Text != "" || len(expression.Allowed) != 0 || expression.Fallback != "" {
			return errors.New("local expression has invalid or irrelevant fields")
		}
	case ExprBinary:
		if _, ok := operations[expression.Operation]; !ok || !priorChild(expression.Left) || !priorChild(expression.Right) || expression.Name != "" || expression.Int != 0 || expression.Bool || expression.HoleID != "" || expression.Text != "" || len(expression.Allowed) != 0 || expression.Fallback != "" {
			return errors.New("binary expression has invalid operation, children, or extra fields")
		}
	case ExprHole:
		if !validHoleID(expression.HoleID) || len(expression.Text) == 0 || len(expression.Text) > maxHoleText || !utf8.ValidString(expression.Text) || !priorChild(expression.Left) || !priorChild(expression.Right) || expression.Name != "" || expression.Int != 0 || expression.Bool || expression.Operation != "" || expression.Fallback == "" || len(expression.Allowed) == 0 || len(expression.Allowed) > len(operations) {
			return errors.New("hole has invalid ID, text, candidates, children, or extra fields")
		}
		allowedSeen := make(map[string]bool, len(expression.Allowed))
		for _, candidate := range expression.Allowed {
			if _, ok := operations[candidate]; !ok || allowedSeen[candidate] {
				return errors.New("hole candidates must be unique supported operations")
			}
			allowedSeen[candidate] = true
		}
		if !allowedSeen[expression.Fallback] {
			return errors.New("hole fallback must be one of its allowed operations")
		}
	default:
		return fmt.Errorf("unknown expression kind %q", expression.Kind)
	}
	return nil
}

func validateStatementShape(statement Stmt, index int) error {
	switch statement.Kind {
	case StmtLet, StmtAssign:
		if len(statement.Then) != 0 || len(statement.Else) != 0 {
			return fmt.Errorf("statement %d has branch lists outside an if", index)
		}
	case StmtIf:
		if statement.Name != "" {
			return fmt.Errorf("if statement %d has a name", index)
		}
	case StmtReturn:
		if statement.Name != "" || len(statement.Then) != 0 || len(statement.Else) != 0 {
			return fmt.Errorf("return statement %d has irrelevant fields", index)
		}
	default:
		return fmt.Errorf("statement %d has unknown kind %q", index, statement.Kind)
	}
	return nil
}

func sameHole(left, right HoleSpec) bool {
	if left.ID != right.ID || left.Text != right.Text || left.LeftType != right.LeftType || left.RightType != right.RightType || left.ResultType != right.ResultType || left.Fallback != right.Fallback || len(left.Allowed) != len(right.Allowed) {
		return false
	}
	for i := range left.Allowed {
		if left.Allowed[i] != right.Allowed[i] {
			return false
		}
	}
	return true
}

func (scope *typeScope) lookup(name string) (decision.ValueType, bool) {
	for current := scope; current != nil; current = current.parent {
		if valueType, ok := current.values[name]; ok {
			return valueType, true
		}
	}
	return "", false
}

func validLocalName(name string) bool {
	return validIdentifier(name) && name != "input"
}

func validIdentifier(name string) bool {
	if !token.IsIdentifier(name) || name == "_" {
		return false
	}
	for index := 0; index < len(name); index++ {
		value := name[index]
		if index == 0 {
			if !((value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || value == '_') {
				return false
			}
			continue
		}
		if !((value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9') || value == '_') {
			return false
		}
	}
	return true
}

func validHoleID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		b := id[i]
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-' || b == '.') {
			return false
		}
	}
	return true
}

func hasControl(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func clonePlan(plan Plan) Plan {
	copyPlan := plan
	copyPlan.Expressions = append([]Expr(nil), plan.Expressions...)
	for index := range copyPlan.Expressions {
		copyPlan.Expressions[index].Allowed = append([]string(nil), plan.Expressions[index].Allowed...)
	}
	copyPlan.Statements = append([]Stmt(nil), plan.Statements...)
	for index := range copyPlan.Statements {
		copyPlan.Statements[index].Then = append([]int(nil), plan.Statements[index].Then...)
		copyPlan.Statements[index].Else = append([]int(nil), plan.Statements[index].Else...)
	}
	copyPlan.Root = append([]int(nil), plan.Root...)
	return copyPlan
}

func checkGoSource(source []byte) error {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "generated.go", source, parser.AllErrors)
	if err != nil {
		return err
	}
	_, err = (&types.Config{Importer: importer.Default()}).Check("generated", fileSet, []*ast.File{file}, nil)
	return err
}

type sourceDialect int

const (
	renderGo sourceDialect = iota
	renderGooo
)

func renderSequence(plan Plan, choices map[string]string, indices []int, dialect sourceDialect, depth int) (string, error) {
	var output strings.Builder
	for _, index := range indices {
		statement := plan.Statements[index]
		indent := strings.Repeat("\t", depth+1)
		switch statement.Kind {
		case StmtLet:
			expression, err := renderExpression(plan, choices, statement.Expr, dialect)
			if err != nil {
				return "", err
			}
			declaration := "var "
			if dialect == renderGooo {
				declaration = "let "
			}
			fmt.Fprintf(&output, "%s%s%s = %s\n", indent, declaration, statement.Name, expression)
		case StmtAssign:
			expression, err := renderExpression(plan, choices, statement.Expr, dialect)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&output, "%s%s = %s\n", indent, statement.Name, expression)
		case StmtIf:
			condition, err := renderExpression(plan, choices, statement.Expr, dialect)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&output, "%sif %s {\n", indent, condition)
			thenBody, err := renderSequence(plan, choices, statement.Then, dialect, depth+1)
			if err != nil {
				return "", err
			}
			output.WriteString(thenBody)
			fmt.Fprintf(&output, "%s}", indent)
			if len(statement.Else) > 0 {
				output.WriteString(" else {\n")
				elseBody, err := renderSequence(plan, choices, statement.Else, dialect, depth+1)
				if err != nil {
					return "", err
				}
				output.WriteString(elseBody)
				fmt.Fprintf(&output, "%s}\n", indent)
			} else {
				output.WriteByte('\n')
			}
		case StmtReturn:
			expression, err := renderExpression(plan, choices, statement.Expr, dialect)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&output, "%sreturn %s\n", indent, expression)
		default:
			return "", fmt.Errorf("cannot render statement kind %q", statement.Kind)
		}
	}
	return output.String(), nil
}

func renderExpression(plan Plan, choices map[string]string, index int, dialect sourceDialect) (string, error) {
	expression := plan.Expressions[index]
	switch expression.Kind {
	case ExprInput:
		return "input", nil
	case ExprInt:
		literal := strconv.FormatInt(expression.Int, 10)
		if dialect == renderGo {
			return "int64(" + literal + ")", nil
		}
		return literal, nil
	case ExprBool:
		return strconv.FormatBool(expression.Bool), nil
	case ExprLocal:
		return expression.Name, nil
	case ExprBinary, ExprHole:
		left, err := renderExpression(plan, choices, expression.Left, dialect)
		if err != nil {
			return "", err
		}
		right, err := renderExpression(plan, choices, expression.Right, dialect)
		if err != nil {
			return "", err
		}
		operation := expression.Operation
		if expression.Kind == ExprHole {
			operation = choices[expression.HoleID]
		}
		operator, ok := operations[operation]
		if !ok {
			return "", fmt.Errorf("unsupported operation %q", operation)
		}
		return "(" + left + " " + operator + " " + right + ")", nil
	default:
		return "", fmt.Errorf("cannot render expression kind %q", expression.Kind)
	}
}

func goType(valueType decision.ValueType) string {
	if valueType == decision.TypeBool {
		return "bool"
	}
	return "int64"
}

type runtimeScope struct {
	parent *runtimeScope
	values map[string]Value
}

func (scope *runtimeScope) lookup(name string) (Value, *runtimeScope, bool) {
	for current := scope; current != nil; current = current.parent {
		if value, ok := current.values[name]; ok {
			return value, current, true
		}
	}
	return Value{}, nil, false
}

func evalExpression(plan Plan, choices map[string]string, index int, scope *runtimeScope) (Value, error) {
	expression := plan.Expressions[index]
	switch expression.Kind {
	case ExprInput, ExprLocal:
		value, _, ok := scope.lookup(expression.Name)
		if !ok {
			return Value{}, fmt.Errorf("runtime name %q is not initialized", expression.Name)
		}
		return value, nil
	case ExprInt:
		return Value{Type: decision.TypeInt, Int: expression.Int}, nil
	case ExprBool:
		return Value{Type: decision.TypeBool, Bool: expression.Bool}, nil
	case ExprBinary, ExprHole:
		left, err := evalExpression(plan, choices, expression.Left, scope)
		if err != nil {
			return Value{}, err
		}
		operation := expression.Operation
		if expression.Kind == ExprHole {
			operation = choices[expression.HoleID]
		}
		if operation == "and" && !left.Bool {
			return Value{Type: decision.TypeBool, Bool: false}, nil
		}
		if operation == "or" && left.Bool {
			return Value{Type: decision.TypeBool, Bool: true}, nil
		}
		right, err := evalExpression(plan, choices, expression.Right, scope)
		if err != nil {
			return Value{}, err
		}
		switch operation {
		case "add":
			return Value{Type: decision.TypeInt, Int: left.Int + right.Int}, nil
		case "subtract":
			return Value{Type: decision.TypeInt, Int: left.Int - right.Int}, nil
		case "multiply":
			return Value{Type: decision.TypeInt, Int: left.Int * right.Int}, nil
		case "less_than":
			return Value{Type: decision.TypeBool, Bool: left.Int < right.Int}, nil
		case "less_equal":
			return Value{Type: decision.TypeBool, Bool: left.Int <= right.Int}, nil
		case "equal":
			if left.Type == decision.TypeInt {
				return Value{Type: decision.TypeBool, Bool: left.Int == right.Int}, nil
			}
			return Value{Type: decision.TypeBool, Bool: left.Bool == right.Bool}, nil
		case "and":
			return Value{Type: decision.TypeBool, Bool: left.Bool && right.Bool}, nil
		case "or":
			return Value{Type: decision.TypeBool, Bool: left.Bool || right.Bool}, nil
		default:
			return Value{}, fmt.Errorf("unsupported operation %q", operation)
		}
	default:
		return Value{}, fmt.Errorf("unsupported expression kind %q", expression.Kind)
	}
}

func executeSequence(plan Plan, choices map[string]string, indices []int, scope *runtimeScope) (Value, bool, error) {
	for _, index := range indices {
		statement := plan.Statements[index]
		switch statement.Kind {
		case StmtLet:
			value, err := evalExpression(plan, choices, statement.Expr, scope)
			if err != nil {
				return Value{}, false, err
			}
			scope.values[statement.Name] = value
		case StmtAssign:
			value, err := evalExpression(plan, choices, statement.Expr, scope)
			if err != nil {
				return Value{}, false, err
			}
			_, owner, ok := scope.lookup(statement.Name)
			if !ok || owner == nil {
				return Value{}, false, fmt.Errorf("runtime assignment target %q is not initialized", statement.Name)
			}
			owner.values[statement.Name] = value
		case StmtIf:
			condition, err := evalExpression(plan, choices, statement.Expr, scope)
			if err != nil {
				return Value{}, false, err
			}
			branch := statement.Else
			if condition.Bool {
				branch = statement.Then
			}
			branchScope := &runtimeScope{parent: scope, values: make(map[string]Value)}
			value, returned, err := executeSequence(plan, choices, branch, branchScope)
			if err != nil || returned {
				return value, returned, err
			}
		case StmtReturn:
			value, err := evalExpression(plan, choices, statement.Expr, scope)
			return value, true, err
		default:
			return Value{}, false, fmt.Errorf("unsupported statement kind %q", statement.Kind)
		}
	}
	return Value{}, false, nil
}
