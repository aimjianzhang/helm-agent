package command

import (
	"helm-example/pkg/kube"
	"helm-example/pkg/model"
)

func init() {
	Funcs.Add(model.DeploymentInfo, kube.GetDeploymentInfo)
}