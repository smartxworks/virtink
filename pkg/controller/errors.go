package controller

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
)

type reconcileError struct {
	ctrl.Result
}

func (rerr reconcileError) Error() string {
	return fmt.Sprintf("requeue: %v, requeueAfter: %s", rerr.Requeue, rerr.RequeueAfter)
}
