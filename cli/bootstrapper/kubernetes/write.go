/* SPDX-License-Identifier: AGPL-3.0-only
* Copyright (c) Benedict Schlueter
 */

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// writeKubeconfigToDisk writes the kubeconfig to disk.
func (a *Bootstrapper) writeKubeconfigToDisk(ctx context.Context) (err error) {
	file, err := a.vmAPI.GetKubernetesConfig(ctx)
	if err != nil {
		return err
	}
	a.adminConf = file
	adminConfigFile, err := os.Create("admin.conf")
	defer func() {
		err = errors.Join(err, adminConfigFile.Close())
	}()
	if err != nil {
		return fmt.Errorf("failed to create admin config file %v: %w", adminConfigFile.Name(), err)
	}

	if _, err := adminConfigFile.Write(file); err != nil {
		return fmt.Errorf("writing kubeadm init yaml config %v failed: %w", adminConfigFile.Name(), err)
	}
	return os.Setenv("KUBECONFIG", "admin.conf")
}
