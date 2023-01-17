package helm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
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
	path, err := iCli.LocateChart(config.CiliumPath, settings)
	if err != nil {
		return err
	}
	logger.Info("helm chart located", zap.String("path", path))
	chartFile, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sha256 := sha256.Sum256(chartFile)
	sha256String := fmt.Sprintf("%x", sha256)
	logger.Info("sha256", zap.String("hash", sha256String))
	if sha256String != config.CiliumHash {
		return fmt.Errorf("sha256 mismatch: got %s, expected %s", sha256String, config.CiliumHash)
	}
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
