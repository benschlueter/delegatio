/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: AGPL-3.0-only
*/

/*
Package terraform handles creation/destruction of a Constellation cluster using Terraform.

Since Terraform does not provide a stable Go API, we use the `terraform-exec` package to interact with Terraform.

The Terraform templates are located in the "terraform" subdirectory. The templates are embedded into the CLI binary using `go:embed`.
On use the relevant template is extracted to the working directory and the user customized variables are written to a `terraform.tfvars` file.
*/
package terraform

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/file"
	"github.com/hashicorp/go-version"
	install "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/spf13/afero"
)

const (
	tfVersion         = ">= 1.4.6"
	terraformVarsFile = "terraform.tfvars"
)

// ErrTerraformWorkspaceExistsWithDifferentVariables is returned when existing Terraform files differ from the version the CLI wants to extract.
var ErrTerraformWorkspaceExistsWithDifferentVariables = errors.New("creating cluster: a Terraform workspace already exists with different variables")

// Client manages interaction with Terraform.
type Client struct {
	tf tfInterface

	file       file.Handler
	workingDir string
	remove     func()
}

// New sets up a new Client for Terraform.
func New(ctx context.Context, workingDir string) (*Client, error) {
	file := file.NewHandler(afero.NewOsFs())
	if err := file.MkdirAll(workingDir); err != nil {
		return nil, err
	}
	tf, remove, err := GetExecutable(ctx, workingDir)
	if err != nil {
		return nil, err
	}

	return &Client{
		tf:         tf,
		remove:     remove,
		file:       file,
		workingDir: workingDir,
	}, nil
}

// Show reads the default state path and outputs the state.
func (c *Client) Show(ctx context.Context) (*tfjson.State, error) {
	return c.tf.Show(ctx)
}

// PrepareWorkspace prepares a Terraform workspace for a Constellation cluster.
func (c *Client) PrepareWorkspace(path string, vars Variables) error {
	if err := prepareWorkspace(path, c.file, c.workingDir); err != nil {
		return fmt.Errorf("prepare workspace: %w", err)
	}

	return c.writeVars(vars)
}

// CreateCluster creates a Constellation cluster using Terraform.
func (c *Client) CreateCluster(ctx context.Context, logLevel LogLevel, targets ...string) (CreateOutput, error) {
	if err := c.setLogLevel(logLevel); err != nil {
		return CreateOutput{}, fmt.Errorf("set terraform log level %s: %w", logLevel.String(), err)
	}

	if err := c.tf.Init(ctx); err != nil {
		return CreateOutput{}, fmt.Errorf("terraform init: %w", err)
	}

	opts := []tfexec.ApplyOption{}
	for _, target := range targets {
		opts = append(opts, tfexec.Target(target))
	}

	if err := c.tf.Apply(ctx, opts...); err != nil {
		return CreateOutput{}, fmt.Errorf("terraform apply: %w", err)
	}

	tfState, err := c.tf.Show(ctx)
	if err != nil {
		return CreateOutput{}, fmt.Errorf("terraform show: %w", err)
	}

	ipOutput, ok := tfState.Values.Outputs["ip"]
	if !ok {
		return CreateOutput{}, errors.New("no IP output found")
	}
	ip, ok := ipOutput.Value.(string)
	if !ok {
		return CreateOutput{}, errors.New("invalid type in IP output: not a string")
	}

	uidOutput, ok := tfState.Values.Outputs["uid"]
	if !ok {
		return CreateOutput{}, errors.New("no uid output found")
	}
	uid, ok := uidOutput.Value.(string)
	if !ok {
		return CreateOutput{}, errors.New("invalid type in uid output: not a string")
	}

	return CreateOutput{
		IP:  ip,
		UID: uid,
	}, nil
}

// CreateOutput contains the Terraform output values of a cluster creation.
type CreateOutput struct {
	IP     string
	Secret string
	UID    string
	// AttestationURL is the URL of the attestation provider.
	// It is only set if the cluster is created on Azure.
	AttestationURL string
}

// IAMOutput contains the output information of the Terraform IAM operations.
type IAMOutput struct {
	GCP GCPIAMOutput
}

// GCPIAMOutput contains the output information of the Terraform IAM operation on GCP.
type GCPIAMOutput struct {
	SaKey string
}

// CreateIAMConfig creates an IAM configuration using Terraform.
func (c *Client) CreateIAMConfig(ctx context.Context, logLevel LogLevel) (IAMOutput, error) {
	if err := c.setLogLevel(logLevel); err != nil {
		return IAMOutput{}, fmt.Errorf("set terraform log level %s: %w", logLevel.String(), err)
	}

	if err := c.tf.Init(ctx); err != nil {
		return IAMOutput{}, err
	}

	if err := c.tf.Apply(ctx); err != nil {
		return IAMOutput{}, err
	}

	tfState, err := c.tf.Show(ctx)
	if err != nil {
		return IAMOutput{}, err
	}

	saKeyOutputRaw, ok := tfState.Values.Outputs["sa_key"]
	if !ok {
		return IAMOutput{}, errors.New("no service account key output found")
	}
	saKeyOutput, ok := saKeyOutputRaw.Value.(string)
	if !ok {
		return IAMOutput{}, errors.New("invalid type in service account key output: not a string")
	}
	return IAMOutput{
		GCP: GCPIAMOutput{
			SaKey: saKeyOutput,
		},
	}, nil
}

// Plan determines the diff that will be applied by Terraform. The plan output is written to the planFile.
// If there is a diff, the returned bool is true. Otherwise, it is false.
func (c *Client) Plan(ctx context.Context, logLevel LogLevel, planFile string, targets ...string) (bool, error) {
	if err := c.setLogLevel(logLevel); err != nil {
		return false, fmt.Errorf("set terraform log level %s: %w", logLevel.String(), err)
	}

	if err := c.tf.Init(ctx); err != nil {
		return false, fmt.Errorf("terraform init: %w", err)
	}

	opts := []tfexec.PlanOption{
		tfexec.Out(planFile),
	}
	for _, target := range targets {
		opts = append(opts, tfexec.Target(target))
	}
	return c.tf.Plan(ctx, opts...)
}

// ShowPlan formats the diff in planFilePath and writes it to the specified output.
func (c *Client) ShowPlan(ctx context.Context, logLevel LogLevel, planFilePath string, output io.Writer) error {
	if err := c.setLogLevel(logLevel); err != nil {
		return fmt.Errorf("set terraform log level %s: %w", logLevel.String(), err)
	}

	planResult, err := c.tf.ShowPlanFileRaw(ctx, planFilePath)
	if err != nil {
		return fmt.Errorf("terraform show plan: %w", err)
	}

	_, err = output.Write([]byte(planResult))
	if err != nil {
		return fmt.Errorf("write plan output: %w", err)
	}

	return nil
}

// Destroy destroys Terraform-created cloud resources.
func (c *Client) Destroy(ctx context.Context, logLevel LogLevel) error {
	if err := c.setLogLevel(logLevel); err != nil {
		return fmt.Errorf("set terraform log level %s: %w", logLevel.String(), err)
	}

	if err := c.tf.Init(ctx); err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}
	return c.tf.Destroy(ctx)
}

// RemoveInstaller removes the Terraform installer, if it was downloaded for this command.
func (c *Client) RemoveInstaller() {
	c.remove()
}

// CleanUpWorkspace removes terraform files from the current directory.
func (c *Client) CleanUpWorkspace() error {
	return cleanUpWorkspace(c.file, c.workingDir)
}

// GetExecutable returns a Terraform executable either from the local filesystem,
// or downloads the latest version fulfilling the version constraint.
func GetExecutable(ctx context.Context, workingDir string) (terraform *tfexec.Terraform, remove func(), err error) {
	inst := install.NewInstaller()

	version, err := version.NewConstraint(tfVersion)
	if err != nil {
		return nil, nil, err
	}

	downloadVersion := &releases.LatestVersion{
		Product:     product.Terraform,
		Constraints: version,
	}
	localVersion := &fs.Version{
		Product:     product.Terraform,
		Constraints: version,
	}

	execPath, err := inst.Ensure(ctx, []src.Source{localVersion, downloadVersion})
	if err != nil {
		return nil, nil, err
	}

	tf, err := tfexec.NewTerraform(workingDir, execPath)

	return tf, func() { _ = inst.Remove(context.Background()) }, err
}

// writeVars tries to write the Terraform variables file or, if it exists, checks if it is the same as we are expecting.
func (c *Client) writeVars(vars Variables) error {
	if vars == nil {
		return errors.New("creating cluster: vars is nil")
	}

	pathToVarsFile := filepath.Join(c.workingDir, terraformVarsFile)
	if err := c.file.Write(pathToVarsFile, []byte(vars.String())); errors.Is(err, afero.ErrFileExists) {
		// If a variables file already exists, check if it's the same as we're expecting, so we can continue using it.
		varsContent, err := c.file.Read(pathToVarsFile)
		if err != nil {
			return fmt.Errorf("read variables file: %w", err)
		}
		if vars.String() != string(varsContent) {
			return ErrTerraformWorkspaceExistsWithDifferentVariables
		}
	} else if err != nil {
		return fmt.Errorf("write variables file: %w", err)
	}

	return nil
}

// setLogLevel sets the log level for Terraform.
func (c *Client) setLogLevel(logLevel LogLevel) error {
	if logLevel.String() != "" {
		if err := c.tf.SetLog(logLevel.String()); err != nil {
			return fmt.Errorf("set log level %s: %w", logLevel.String(), err)
		}
		if err := c.tf.SetLogPath(filepath.Join("..", config.TerraformLogFile)); err != nil {
			return fmt.Errorf("set log path: %w", err)
		}
	}
	return nil
}

type tfInterface interface {
	Apply(context.Context, ...tfexec.ApplyOption) error
	Destroy(context.Context, ...tfexec.DestroyOption) error
	Init(context.Context, ...tfexec.InitOption) error
	Show(context.Context, ...tfexec.ShowOption) (*tfjson.State, error)
	Plan(ctx context.Context, opts ...tfexec.PlanOption) (bool, error)
	ShowPlanFileRaw(ctx context.Context, planPath string, opts ...tfexec.ShowOption) (string, error)
	SetLog(level string) error
	SetLogPath(path string) error
}
