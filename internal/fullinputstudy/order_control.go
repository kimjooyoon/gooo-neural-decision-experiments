package fullinputstudy

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

type OrderControl struct {
	Schema          string      `json:"schema"`
	Inputs          [2]string   `json:"complete_feature_inputs"`
	PositionedSHA   [2]string   `json:"positioned_feature_sha256"`
	BagSHA          [2]string   `json:"bag_feature_sha256"`
	Arguments       [3]int64    `json:"finite_arguments"`
	Outputs         [2][3]int64 `json:"reference_arithmetic_outputs"`
	BagAliases      bool        `json:"bag_aliases_distinct_operation_order"`
	PositionDiffers bool        `json:"positioned_features_distinguish_pair"`
	NativeCalls     int         `json:"native_executions"`
	Scope           string      `json:"scope"`
}

// MakeOrderControl records an explicit limit of byte-fragment bag features.
// Arithmetic outputs are reference calculations, with zero native executions.
func MakeOrderControl() (OrderControl, error) {
	c := OrderControl{Schema: "gooo/full-input-order-control/v1", Arguments: [3]int64{-2, 0, 3},
		Scope: "Synthetic feature probe with a shared valid source header; reference arithmetic illustrates operation order. No compiled Gooo execution or trained prediction."}
	var fields [64]byte
	fields[0], fields[5] = 128, 128
	for i, text := range [2]string{"Step: add one. Step: multiply by two. Step: end.", "Step: multiply by two. Step: add one. Step: end."} {
		var err error
		c.Inputs[i], err = decision.EncodeSemanticContextInput(fields, text)
		if err != nil {
			return c, err
		}
		var v3, v4 [256]float32
		if err = decision.SemanticContextFeaturesInto(c.Inputs[i], &v3); err != nil {
			return c, err
		}
		if err = decision.SemanticContextBagFeaturesInto(c.Inputs[i], &v4); err != nil {
			return c, err
		}
		c.PositionedSHA[i], c.BagSHA[i] = featureSHA(v3), featureSHA(v4)
	}
	for i, x := range c.Arguments {
		c.Outputs[0][i], c.Outputs[1][i] = (x+1)*2, x*2+1
	}
	c.BagAliases = c.BagSHA[0] == c.BagSHA[1]
	c.PositionDiffers = c.PositionedSHA[0] != c.PositionedSHA[1]
	if !c.BagAliases || !c.PositionDiffers || c.Outputs[0] == c.Outputs[1] {
		return c, errors.New("frozen order-control relation changed")
	}
	return c, nil
}

func featureSHA(values [256]float32) string {
	var raw [1024]byte
	for i, v := range values {
		binary.LittleEndian.PutUint32(raw[i*4:], math.Float32bits(v))
	}
	return threecohort.SHA(raw[:])
}
