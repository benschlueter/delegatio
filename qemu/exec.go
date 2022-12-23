package qemu

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/benschlueter/delegatio/core/config"
	"github.com/benschlueter/delegatio/core/vmapi/vmproto"
	"github.com/benschlueter/delegatio/qemu/definitions"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"libvirt.org/go/libvirt"

	kubeadm "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
)

type qemuExecResponse struct {
	Return struct {
		Pid int `json:"pid"`
	} `json:"return"`
}

type qemuStatusResponse struct {
	Return struct {
		Exitcode int    `json:"exitcode,omitempty"`
		OutData  string `json:"out-data,omitempty"`
		Exited   bool   `json:"exited,omitempty"`
		ErrData  string `json:"err-data,omitempty"`
	} `json:"return,omitempty"`
}

func (l *LibvirtInstance) waitForCompletion(ctx context.Context, pid int, domain *libvirt.Domain) (response *qemuStatusResponse, err error) {
	response = &qemuStatusResponse{}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			result, err := domain.QemuAgentCommand(
				fmt.Sprintf(`
				{
					"execute": "guest-exec-status",
					"arguments": {
						"pid": %d
					}
					}`, pid),
				libvirt.DOMAIN_QEMU_AGENT_COMMAND_BLOCK, 0)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal([]byte(result), response); err != nil {
				return nil, err
			}
			if response.Return.Exited {
				return response, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (l *LibvirtInstance) JoinCluster(ctx context.Context, id string, joinToken *kubeadm.BootstrapTokenDiscovery) (err error) {
	domain, err := l.conn.LookupDomainByName(id)
	if err != nil {
		return err
	}
	l.log.Info("executing kubeadm joins")
	result, err := domain.QemuAgentCommand(
		fmt.Sprintf(`
		{
			"execute": "guest-exec",
			"arguments": {
				"path": "/usr/bin/kubeadm",
				"arg": ["join", "%s", "--token", "%s", "--discovery-token-ca-cert-hash", "%s"],
				"capture-output": true
			}
		}`, joinToken.APIServerEndpoint, joinToken.Token, joinToken.CACertHashes[0]),
		120, 0)
	if err != nil {
		return
	}
	l.log.Info("kubeadm returned")
	var response qemuExecResponse
	json.Unmarshal([]byte(result), &response)

	l.log.Info("wait for completion")

	stateResponse, err := l.waitForCompletion(ctx, response.Return.Pid, domain)
	if err != nil {
		return err
	}
	if stateResponse.Return.Exitcode == 0 {
		return nil
	}
	err = fmt.Errorf("error during 'kubeadm join': %s", stateResponse.Return.ErrData)
	return
}

func (l *LibvirtInstance) InitializeKubernetesQemuGuestAgent(ctx context.Context) (output string, err error) {
	domain, err := l.conn.LookupDomainByName("delegatio-0")
	if err != nil {
		return
	}
	l.log.Info("executing kubeadm")
	result, err := domain.QemuAgentCommand(
		`
		{
			"execute": "guest-exec",
			"arguments": {
				"path": "/usr/bin/kubeadm",
				"arg": ["init"],
				"capture-output": true
			}
		}`,
		120, 0)
	if err != nil {
		return
	}
	l.log.Info("kubeadm init scheduled")
	var response qemuExecResponse
	json.Unmarshal([]byte(result), &response)

	stateResponse, err := l.waitForCompletion(ctx, response.Return.Pid, domain)
	if err != nil {
		return
	}
	l.log.Info("kubeadm init finished")
	sDec, err := base64.StdEncoding.DecodeString(stateResponse.Return.ErrData)
	if err != nil {
		return
	}
	l.log.Info("errors during kubeadm", zap.String("errors", string(sDec)))
	sDec, err = base64.StdEncoding.DecodeString(stateResponse.Return.OutData)
	if err != nil {
		return
	}
	l.log.Info("kubeadm init response", zap.String("response", string(sDec)))
	if stateResponse.Return.Exitcode == 0 {
		return string(sDec), nil
	}
	err = fmt.Errorf("error during 'kubeadm init': %s", stateResponse.Return.ErrData)
	return
}

func (l *LibvirtInstance) InitializeKubernetesgRPC(ctx context.Context) (output string, err error) {
	qemuNetwork, err := l.conn.LookupNetworkByName(definitions.NetworkName)
	if err != nil {
		return
	}
	leases, err := qemuNetwork.GetDHCPLeases()
	if err != nil {
		return
	}
	if len(leases) == 0 {
		return "", fmt.Errorf("no vms in network")
	}
	ip := leases[0].IPaddr

	conn, err := grpc.DialContext(ctx, net.JoinHostPort(ip, config.PublicAPIport))
	if err != nil {
		return "", err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	resp, err := client.ExecCommand(ctx, &vmproto.ExecCommandRequest{
		Command: "/usr/bin/kubeadm init",
	})
	if err != nil {
		return
	}
	return resp.Output, err
}
