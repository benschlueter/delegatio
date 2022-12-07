package qemu

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/google/shlex"
	"go.uber.org/zap"
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

func (l *LibvirtInstance) InitializeKubernetes(ctx context.Context) (output string, err error) {
	domain, err := l.conn.LookupDomainByName("delegatio-0")
	if err != nil {
		return
	}
	// needed because something in the network stack is not ready
	// can probably be fixed by another image without NetworkManager
	time.Sleep(30 * time.Second)
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

func (l *LibvirtInstance) ParseKubeadmOutput(data string) (string, error) {
	stdoutStr := string(data)
	indexKubeadmJoin := strings.Index(stdoutStr, "kubeadm join")
	if indexKubeadmJoin < 0 {
		return "", errors.New("kubeadm init did not return join command")
	}

	joinCommand := strings.ReplaceAll(stdoutStr[indexKubeadmJoin:], "\\\n", " ")
	// `kubeadm init` returns the two join commands, each broken up into two lines with backslash + newline in between.
	// The following functions assume that stdoutStr[indexKubeadmJoin:] look like the following string.

	// -----------------------------------------------------------------------------------------------
	// --- When modifying the kubeadm.InitConfiguration make sure that this assumption still holds ---
	// -----------------------------------------------------------------------------------------------

	// "kubeadm join 127.0.0.1:16443 --token vlhjr4.9l6lhek0b9v65m67 \
	//	--discovery-token-ca-cert-hash sha256:2b5343a162e31b70602e3cab3d87189dc10431e869633c4db63c3bfcd038dee6 \
	//	--control-plane
	//
	// Then you can join any number of worker nodes by running the following on each as root:
	//
	// kubeadm join 127.0.0.1:16443 --token vlhjr4.9l6lhek0b9v65m67 \
	//  --discovery-token-ca-cert-hash sha256:2b5343a162e31b70602e3cab3d87189dc10431e869633c4db63c3bfcd038dee6"

	// Splits the string into a slice, where earch slice-element contains one line from the previous string
	splittedJoinCommand := strings.SplitN(joinCommand, "\n", 2)
	return splittedJoinCommand[0], nil
}

func (l *LibvirtInstance) ParseJoinCommand(joinCommand string) (*kubeadm.BootstrapTokenDiscovery, error) {
	// Format:
	// kubeadm join [API_SERVER_ENDPOINT] --token [TOKEN] --discovery-token-ca-cert-hash [DISCOVERY_TOKEN_CA_CERT_HASH] --control-plane

	// split and verify that this is a kubeadm join command
	argv, err := shlex.Split(joinCommand)
	if err != nil {
		return nil, fmt.Errorf("kubadm join command could not be tokenized: %v", joinCommand)
	}
	if len(argv) < 3 {
		return nil, fmt.Errorf("kubadm join command is too short: %v", argv)
	}
	if argv[0] != "kubeadm" || argv[1] != "join" {
		return nil, fmt.Errorf("not a kubeadm join command: %v", argv)
	}

	result := kubeadm.BootstrapTokenDiscovery{APIServerEndpoint: argv[2]}

	var caCertHash string
	// parse flags
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.StringVar(&result.Token, "token", "", "")
	flags.StringVar(&caCertHash, "discovery-token-ca-cert-hash", "", "")
	flags.Bool("control-plane", false, "")
	if err := flags.Parse(argv[3:]); err != nil {
		return nil, fmt.Errorf("parsing flag arguments failed: %v %w", argv, err)
	}

	if result.Token == "" {
		return nil, fmt.Errorf("missing flag argument token: %v", argv)
	}
	if caCertHash == "" {
		return nil, fmt.Errorf("missing flag argument discovery-token-ca-cert-hash: %v", argv)
	}
	result.CACertHashes = []string{caCertHash}

	return &result, nil
}
