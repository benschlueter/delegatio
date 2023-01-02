package helm

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

func Install(ctx context.Context, logger *zap.Logger, name, namespace string) error {
	chartPath := "/tmp/my-chart-0.1.0.tgz"
	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	settings := cli.New()
	settings.KubeConfig = "admin.conf"
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "Secret", func(format string, v ...interface{}) {
		logger.Info(fmt.Sprintf(format, v))
	}); err != nil {
		return err
	}

	iCli := action.NewInstall(actionConfig)
	iCli.Namespace = namespace
	iCli.ReleaseName = name
	rel, err := iCli.Run(chart, nil)
	if err != nil {
		return err
	}
	logger.Info("installed helm release", zap.String("name", rel.Name))
	return nil
}
