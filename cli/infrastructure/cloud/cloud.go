/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Edgeless Systems GmbH
 * Copyright (c) Benedict Schlueter
 */

package terraform

import (
	"context"
	"errors"
	"fmt"
	"path"

	tfjson "github.com/hashicorp/terraform-json"
)

type terraformClient interface {
	PrepareWorkspace(path string, input Variables) error
	CreateCluster(ctx context.Context, logLevel LogLevel, targets ...string) (CreateOutput, error)
	CreateIAMConfig(ctx context.Context, logLevel LogLevel) (IAMOutput, error)
	Destroy(ctx context.Context, logLevel LogLevel) error
	CleanUpWorkspace() error
	RemoveInstaller()
	Show(ctx context.Context) (*tfjson.State, error)
}

// rollbacker does a rollback.
type rollbacker interface {
	rollback(ctx context.Context, logLevel LogLevel) error
}

// rollbackOnError calls rollback on the rollbacker if the handed error is not nil,
// and writes logs to the writer w.
func rollbackOnError(onErr *error, roll rollbacker, logLevel LogLevel) {
	if *onErr == nil {
		return
	}
	fmt.Printf("An error occurred: %s\n", *onErr)
	fmt.Println("Attempting to roll back.")
	if err := roll.rollback(context.Background(), logLevel); err != nil {
		*onErr = errors.Join(*onErr, fmt.Errorf("on rollback: %w", err)) // TODO(katexochen): print the error, or return it?
		return
	}
	fmt.Println("Rollback succeeded.")
}

type rollbackerTerraform struct {
	client terraformClient
}

func (r *rollbackerTerraform) rollback(ctx context.Context, logLevel LogLevel) error {
	if err := r.client.Destroy(ctx, logLevel); err != nil {
		return err
	}
	return r.client.CleanUpWorkspace()
}

func CreateGCP(ctx context.Context, cl terraformClient) (retErr error) {
	tfLogLevel := LogLevelDebug
	vars := GCPClusterVariables{
		CommonVariables: CommonVariables{
			Name:               "delegatiooooo",
			CountControlPlanes: 1,
			CountWorkers:       1,
			StateDiskSizeGB:    20,
		},
		Project:         "delegatio",
		Region:          "europe-west6",
		Zone:            "europe-west6-a",
		CredentialsFile: "/home/bschlueter/University/Github/delegatio/build/delegatio.json",
		InstanceType:    "g1-small",
		StateDiskType:   "pd-standard",
		ImageID:         "https://www.googleapis.com/compute/v1/projects/delegatio/global/images/gcp-0-0-0-test",
		Debug:           true,
	}

	if err := cl.PrepareWorkspace(path.Join("terraform", "gcp"), &vars); err != nil {
		return err
	}

	defer rollbackOnError(&retErr, &rollbackerTerraform{client: cl}, tfLogLevel)
	tfOutput, err := cl.CreateCluster(ctx, tfLogLevel)
	if err != nil {
		return err
	}
	fmt.Printf(tfOutput.IP)
	return nil
}
