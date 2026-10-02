package main

import (
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

func jointParts(v jointcohort.View) ([2]string, error) { return jointdecision.Parts(v.JointInput) }
