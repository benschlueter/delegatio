package helm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"time"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/k8sapi"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

// Installer is the struct used to install helm charts.
type Installer struct {
	logger      *zap.Logger
	releaseName string
	namespace   string
	path        string
	sha256      string
	configValue map[string]interface{}
}

// NewHelmInstaller returns a new helm installer.
func NewHelmInstaller(logger *zap.Logger, releaseName, namespace, path, sha256 string, configValue map[string]interface{}) *Installer {
	return &Installer{
		logger:      logger,
		releaseName: releaseName,
		namespace:   namespace,
		path:        path,
		sha256:      sha256,
		configValue: configValue,
	}
}

// Install installs the given helm chart.
func (h *Installer) Install(ctx context.Context) error {
	settings := cli.New()
	kubeconfig, err := k8sapi.GetKubeConfigPath()
	if err != nil {
		return err
	}
	settings.KubeConfig = kubeconfig

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), "kube-system", "secret", func(format string, v ...interface{}) {
		h.logger.Info(fmt.Sprintf(format, v))
	}); err != nil {
		return err
	}

	iCli := action.NewInstall(actionConfig)
	iCli.Timeout = 2 * time.Minute
	iCli.ReleaseName = h.releaseName
	iCli.Namespace = h.namespace
	iCli.CreateNamespace = true
	path, err := iCli.LocateChart(h.path, settings)
	if err != nil {
		return err
	}
	h.logger.Info("helm chart located", zap.String("path", path))
	chartFile, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sha256 := sha256.Sum256(chartFile)
	sha256String := fmt.Sprintf("%x", sha256)
	h.logger.Info("hash", zap.String("sha256", sha256String))
	if sha256String != h.sha256 {
		return fmt.Errorf("sha256 mismatch: got %s, expected %s", sha256String, config.Cilium256Hash)
	}
	chart, err := loader.Load(path)
	if err != nil {
		return err
	}

	rel, err := iCli.RunWithContext(ctx, chart, h.configValue)
	if err != nil {
		return err
	}
	h.logger.Info("installed helm release", zap.String("name", rel.Name))
	return nil
}
