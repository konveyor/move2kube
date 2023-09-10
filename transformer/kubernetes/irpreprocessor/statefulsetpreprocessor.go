package irpreprocessor

import (
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
)

type statefulsetPreprocessor struct {
}

func (sp statefulsetPreprocessor) preprocess(ir irtypes.IR) (irtypes.IR, error) {
	for k, scObj := range ir.Services {
		isStateful := commonqa.Stateful()
		scObj.StatefulSet = isStateful
		ir.Services[k] = scObj
	}

	return ir, nil
}