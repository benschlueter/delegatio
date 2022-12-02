package qemu

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
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

func (l *LibvirtInstance) WaitForCompletion(pid int, domain *libvirt.Domain) (response *qemuStatusResponse, err error) {
	response = &qemuStatusResponse{}
	for {
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
		fmt.Println(result)
		time.Sleep(1 * time.Second)
	}
}

func (l *LibvirtInstance) ExecuteCommand() (err error) {
	domain, err := l.conn.LookupDomainByName("delegatio-0")
	if err != nil {
		return err
	}
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
		return err
	}
	l.log.Info("kubeadm returned")
	fmt.Println(result)
	var response qemuExecResponse
	json.Unmarshal([]byte(result), &response)

	l.log.Info("wait for completion")
	stateResponse, err := l.WaitForCompletion(response.Return.Pid, domain)
	if err != nil {
		return err
	}
	fmt.Println(stateResponse)
	l.log.Info("kubeadm init finished")
	sDec, err := base64.StdEncoding.DecodeString(stateResponse.Return.ErrData)
	if err != nil {
		return err
	}
	l.log.Warn("errors during kubeadm", zap.String("errors", string(sDec)))
	fmt.Println(string(sDec))
	sDec, err = base64.StdEncoding.DecodeString(stateResponse.Return.OutData)
	if err != nil {
		return err
	}
	l.log.Info("kubeadm init response", zap.String("response", string(sDec)))
	fmt.Println(string(sDec))
	if stateResponse.Return.Exitcode == 0 {
		return nil
	}
	return fmt.Errorf("error during 'kubeadm init': %s", stateResponse.Return.ErrData)
}
