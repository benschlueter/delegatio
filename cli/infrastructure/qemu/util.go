/* SPDX-License-Identifier: AGPL-3.0-only
* Copyright (c) Edgeless Systems GmbH
 * Copyright (c) Benedict Schlueter
*/

package qemu

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/benschlueter/delegatio/client/config"
	"github.com/benschlueter/delegatio/client/vmapi/vmproto"
	"github.com/google/shlex"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	kubeadm "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	"libvirt.org/go/libvirt"
)

func (l *LibvirtInstance) uploadBaseImage(ctx context.Context, baseVolume *libvirt.StorageVol) (err error) {
	stream, err := l.Conn.NewStream(libvirt.STREAM_NONBLOCK)
	if err != nil {
		return err
	}
	defer func() { _ = stream.Free() }()
	file, err := os.Open(l.ImagePath)
	if err != nil {
		return fmt.Errorf("error while opening %s: %s", l.ImagePath, err)
	}
	defer func() {
		err = multierr.Append(err, file.Close())
	}()

	fi, err := file.Stat()
	if err != nil {
		return err
	}
	if err := baseVolume.Upload(stream, 0, uint64(fi.Size()), 0); err != nil {
		return err
	}
	transferredBytes := 0
	buffer := make([]byte, 4*1024*1024)

	// Fill the stream with buffer-chunks of the image.
	// Since this can take long we must make this interruptable in case of
	// a context cancellation.
loop:
	for {
		select {
		case <-ctx.Done():
			err := stream.Abort()
			l.Log.Info("context cancel during image upload", zap.Error(err))
			return ctx.Err()
		default:
			_, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				return err
			}
			if err == io.EOF {
				break loop
			}
			num, err := stream.Send(buffer)
			if err != nil {
				return err
			}
			transferredBytes += num
		}
	}
	/* 	if err := stream.Finish(); err != nil {
		return err
	} */
	if transferredBytes < int(fi.Size()) {
		return fmt.Errorf("only send %d out of %d bytes", transferredBytes, fi.Size())
	}
	l.Log.Info("image upload successful", zap.Int64("image size", fi.Size()), zap.Int("transferred bytes", transferredBytes))
	return nil
}

func (l *LibvirtInstance) blockUntilNetworkIsReady(ctx context.Context) error {
	domain, err := l.Conn.LookupDomainByName("delegatio-0")
	if err != nil {
		return err
	}
	defer func() { _ = domain.Free() }()
	for {
		select {
		case <-ctx.Done():
			l.Log.Info("context cancel during waiting for vm init")
			return ctx.Err()
		default:
			iface, err := domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE)
			if err != nil {
				return err
			}
			var ip string
			for _, netInterface := range iface {
				if netInterface.Name == "lo" {
					continue
				}
				for _, addr := range netInterface.Addrs {
					if addr.Type == libvirt.IP_ADDR_TYPE_IPV4 {
						ip = addr.Addr
					}
				}
			}
			if len(ip) > 0 {
				return nil
			}
		}
	}
}

func (l *LibvirtInstance) blockUntilDelegatioAgentIsReady(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	domain, err := l.Conn.LookupDomainByName("delegatio-0")
	if err != nil {
		return err
	}
	defer func() { _ = domain.Free() }()
	iface, err := domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE)
	if err != nil {
		return err
	}
	var ip string
	for _, netInterface := range iface {
		if netInterface.Name == "lo" {
			continue
		}
		for _, addr := range netInterface.Addrs {
			if addr.Type == libvirt.IP_ADDR_TYPE_IPV4 {
				ip = addr.Addr
			}
		}
	}
	if len(ip) == 0 {
		return fmt.Errorf("could not get ip addr of VM %s", "delegatio-0")
	}
	conn, err := grpc.DialContext(ctx, net.JoinHostPort(ip, config.PublicAPIport), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := vmproto.NewAPIClient(conn)
	for {
		select {
		case <-ctx.Done():
			l.Log.Info("context cancel during waiting for vm init")
			return ctx.Err()
		default:
			_, err := client.ExecCommand(ctx, &vmproto.ExecCommandRequest{
				Command: "whoami",
			})
			if err == nil {
				return nil
			}
		}
	}
}

func (l *LibvirtInstance) parseKubeadmOutput(data []byte) (string, error) {
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

func (l *LibvirtInstance) parseJoinCommand(joinCommand string) (*kubeadm.BootstrapTokenDiscovery, error) {
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
