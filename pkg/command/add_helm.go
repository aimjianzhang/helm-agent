package command

import (
	"helm-example/pkg/helm"
	"helm-example/pkg/model"
)

func init() {
	Funcs.Add(model.InstallRelease, helm.InstallRelease)
}