package helm

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

// Install installs the given helm chart.
func Install(ctx context.Context, logger *zap.Logger, name, apiServerAddr string) error {
	settings := cli.New()
	settings.KubeConfig = "./admin.conf"

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), "kube-system", "secret", func(format string, v ...interface{}) {
		logger.Info(fmt.Sprintf(format, v))
	}); err != nil {
		return err
	}

	iCli := action.NewInstall(actionConfig)
	iCli.Timeout = 2 * time.Minute
	iCli.ReleaseName = "cilium"
	iCli.Namespace = "kube-system"
	iCli.CreateNamespace = true
	path, err := iCli.LocateChart("https://github.com/cilium/charts/raw/master/cilium-1.12.5.tgz", settings)
	if err != nil {
		return err
	}
	logger.Info("helm chart located", zap.String("path", path))
	chart, err := loader.Load(path)
	if err != nil {
		return err
	}
	vals := map[string]interface{}{
		"kubeProxyReplacement": "strict",
		"k8sServicePort":       "6443",
		"k8sServiceHost":       apiServerAddr,
		/* 		"prometheus.enabled":          "true",
		   		"operator.prometheus.enabled": true, */
	}

	rel, err := iCli.RunWithContext(ctx, chart, vals)
	if err != nil {
		return err
	}
	logger.Info("installed helm release", zap.String("name", rel.Name))
	return nil
}
