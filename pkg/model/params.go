package model

type DeploymentParams struct {
	Namespace      string `json:"namespace,omitempty"`
	DeploymentName string `json:"deploymentName,omitempty"`
}

type HelmInstallReleaseParams struct {
	Namespace      string `json:"namespace,omitempty"`
	RepoURL        string `json:"repoUrl,omitempty"`
	RepoUserName   string `json:"repoUserName,omitempty"`
	RepoPassword   string `json:"repoPassword,omitempty"`
	ReleaseName    string `json:"releaseName,omitempty"`
	ChartName      string `json:"chartName,omitempty"`
	ChartVersion   string `json:"chartVersion,omitempty"`
	Values         string `json:"values,omitempty"`
}