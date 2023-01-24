/* SPDX-License-Identifier: AGPL-3.0-only
* Copyright (c) Benedict Schlueter
 */

package configurer

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/multierr"
)

// writeKubeconfigToDisk writes the kubeconfig to disk.
func (a *Configurer) writeKubeconfigToDisk(ctx context.Context) (err error) {
	file, err := a.getKubernetesConfig(ctx)
	if err != nil {
		return err
	}
	adminConfigFile, err := os.Create("admin.conf")
	defer func() {
		err = multierr.Append(err, adminConfigFile.Close())
	}()
	if err != nil {
		return fmt.Errorf("failed to create admin config file %v: %w", adminConfigFile.Name(), err)
	}

	if _, err := adminConfigFile.Write(file); err != nil {
		return fmt.Errorf("writing kubeadm init yaml config %v failed: %w", adminConfigFile.Name(), err)
	}
	return
}
