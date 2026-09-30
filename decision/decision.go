// Package decision exposes the bounded Go-only typed-operation model runtime.
//
// The model returns one of the closed operation labels and, through Decide,
// may turn that label into validated typed binary IR. It does not generate or
// execute arbitrary source code. Model bundles remain separate files loaded by
// Load; they are never embedded in this package.
package decision

import internal "github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"

const (
	MetadataSchema         = internal.MetadataSchema
	TypedBinaryIRSchema    = internal.TypedBinaryIRSchema
	DecisionRequestSchema  = internal.DecisionRequestSchema
	DecisionResponseSchema = internal.DecisionResponseSchema
	InputMaxBytes          = internal.InputMaxBytes
)

type (
	Model            = internal.Model
	Workspace        = internal.Workspace
	Prediction       = internal.Prediction
	DecisionRequest  = internal.DecisionRequest
	DecisionResponse = internal.DecisionResponse
	LabelProbability = internal.LabelProbability
	ValueType        = internal.ValueType
	Identifier       = internal.Identifier
	TypedBinaryIR    = internal.TypedBinaryIR
)

const (
	TypeInt  = internal.TypeInt
	TypeBool = internal.TypeBool
)

// Load reads and validates a model metadata file and its sibling weights file.
// Metadata and packed weights have strict size bounds and the weights digest
// must match the metadata before tensors are decoded.
func Load(metadataPath string) (*Model, error) {
	return internal.Load(metadataPath)
}

// Labels returns a copy of the fixed operation label list. Mutating the result
// cannot change labels used by future calls.
func Labels() []string {
	labels := internal.Labels()
	return append([]string(nil), labels[:]...)
}

// BuildTypedBinary converts one closed operation label and two typed
// identifiers into validated binary IR. It accepts no source fragments.
func BuildTypedBinary(operation string, left, right Identifier) (TypedBinaryIR, error) {
	return internal.BuildTypedBinary(operation, left, right)
}

// AssembleGoExpression renders validated typed binary IR using its fixed
// operator mapping and validated identifier names.
func AssembleGoExpression(ir TypedBinaryIR) (string, error) {
	return internal.AssembleGoExpression(ir)
}

// WorkspaceBytes reports the fixed scratch arrays used by one inference
// worker. Allocate one Workspace per concurrent PredictInto call.
func WorkspaceBytes() int {
	return internal.WorkspaceBytes()
}

// PredictionValueArrayBytes reports the combined fixed logits and probability
// arrays in a Prediction.
func PredictionValueArrayBytes() int {
	return internal.PredictionValueArrayBytes()
}

// PredictionBytes reports the complete caller-owned Prediction struct size on
// the current architecture.
func PredictionBytes() int {
	return internal.PredictionBytes()
}
