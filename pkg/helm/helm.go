package helm

import (
	"encoding/json"
	"fmt"
	"github.com/aimjianzhang/helm/pkg/action"
	"github.com/aimjianzhang/helm/pkg/chart"
	"github.com/aimjianzhang/helm/pkg/chart/loader"
	"github.com/aimjianzhang/helm/pkg/cli"
	values2 "github.com/aimjianzhang/helm/pkg/cli/values"
	"github.com/aimjianzhang/helm/pkg/downloader"
	"github.com/aimjianzhang/helm/pkg/getter"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"helm-example/pkg/model"
	"log"
	"os"
)

func debug(format string, v ...interface{}) {
	format = fmt.Sprintf("[debug] %s\n", format)
	log.Output(2, fmt.Sprintf(format, v...))
}

func InstallRelease(context *model.Context, cmd *model.Packet) ([]*model.Packet, *model.Packet) {
	var helmInstallReleaseParams model.HelmInstallReleaseParams
	err := json.Unmarshal([]byte(cmd.Payload), helmInstallReleaseParams)
	if err != nil {
		return nil, NewResponseReleaseError(cmd.Key, cmd.Type, err.Error())
	}

	chartPathOptions := action.ChartPathOptions{
		RepoURL:  helmInstallReleaseParams.RepoURL,
		Version:  helmInstallReleaseParams.ChartVersion,
		Username: helmInstallReleaseParams.RepoUserName,
		Password: helmInstallReleaseParams.RepoPassword,
	}
	// 获取helm配置信息
	actionConfiguration, settings := getCfg(helmInstallReleaseParams.Namespace)

	// 创建install客户端，可以设置helm install支持的参数
	instClient := action.NewInstall(actionConfiguration)
	// 设置chart仓库地址及版本、认证信息等
	instClient.ChartPathOptions = chartPathOptions
	// 设置安装chart包等待其安装完成
	instClient.Wait = true
	// 需要在那个命名空间下安装
	instClient.Namespace = helmInstallReleaseParams.Namespace
	// release的名称
	instClient.ReleaseName = helmInstallReleaseParams.ReleaseName

	// 查找chart返回完整路径或错误。如果有配置验证chart，将尝试验证
	cp, err := instClient.ChartPathOptions.LocateChart(helmInstallReleaseParams.ChartName, settings)
	if err != nil {
		return nil, NewResponseReleaseError(cmd.Key, cmd.Type, err.Error())
	}

	valuesOptions := getValuesOptions(helmInstallReleaseParams.Values)
	getters := getter.All(settings)
	vals, err := valuesOptions.MergeValues(getters)
	if err != nil {
		return nil, NewResponseReleaseError(cmd.Key, cmd.Type, err.Error())
	}
	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return nil, NewResponseReleaseError(cmd.Key, cmd.Type, err.Error())
	}
	if err = checkIfInstallable(chartRequested); err != nil {
		return nil, NewResponseReleaseError(cmd.Key, cmd.Type, err.Error())
	}
	if chartRequested.Metadata.Deprecated {
		glog.Info("This chart is deprecated")
	}
	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err = action.CheckDependencies(chartRequested, req); err != nil {
			if instClient.DependencyUpdate {
				man := &downloader.Manager{
					ChartPath:        cp,
					Keyring:          instClient.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          getters,
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
					Debug:            settings.Debug,
				}
				if err = man.Update(); err != nil {
					return nil, NewResponseReleaseError(cmd.Key, cmd.Type, err.Error())
				}
				if chartRequested, err = loader.Load(cp); err != nil {
					return nil, NewResponseReleaseError(cmd.Key, cmd.Type, err.Error())
				}
			} else {
				return nil, NewResponseReleaseError(cmd.Key, cmd.Type, err.Error())
			}
		}
	}
	responseRelease, err := instClient.Run(chartRequested, vals)
	if err != nil {
		return nil, NewResponseReleaseError(cmd.Key, cmd.Type, err.Error())
	}
	responseReleaseJson, err := json.Marshal(responseRelease)
	if err != nil {
		return nil, NewResponseReleaseError(cmd.Key, cmd.Type, err.Error())
	}
	return nil, &model.Packet{
		Key:     cmd.Key,
		Type:    cmd.Type,
		Payload: string(responseReleaseJson),
	}
}

func checkIfInstallable(ch *chart.Chart) error {
	switch ch.Metadata.Type {
	case "", "application":
		return nil
	}
	return errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func getValuesOptions(values string) *values2.Options {
	return &values2.Options{
		RequestValues: values,
	}
}

// 获取helm配置信息
func getCfg(namespace string) (*action.Configuration, *cli.EnvSettings) {
	settings := cli.New()
	settings.SetNamespace(namespace)

	actionConfigration := &action.Configuration{}
	helmDriver := os.Getenv("HELM_DRIVER")

	if err := actionConfigration.Init(settings.RESTClientGetter(), settings.Namespace(), helmDriver, debug); err != nil {
		log.Fatal(err)
	}
	return actionConfigration, settings
}
