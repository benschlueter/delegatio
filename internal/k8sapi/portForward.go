/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package k8sapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/transport/spdy"
)

const portForwardProtocolV1Name = "portforward.k8s.io"

// CreatePodPortForward creates a port forward to a pod.
func (k *Client) CreatePodPortForward(ctx context.Context, namespace, podName, podPort string, channel ssh.Channel) error {
	roundTripper, upgrader, err := spdy.RoundTripperFor(k.RestConfig)
	if err != nil {
		k.logger.Error("failed to create round tripper", zap.Error(err))
		return err
	}

	req := k.Client.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace(namespace).SubResource("portforward")

	/* 	hostIP := strings.TrimLeft(k.restClient.Host, "htps:/")
	   	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP} */

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, req.URL())
	connection, _, err := dialer.Dial(portForwardProtocolV1Name)
	if err != nil {
		k.logger.Error("failed to dial kubeapi server", zap.Error(err))
		return err
	}
	defer func() {
		if err := connection.Close(); err != nil {
			k.logger.Error("closing kubeapi connection", zap.Error(err))
		}
	}()

	k.logger.Info("handling forwarding connection", zap.String("pod", podName), zap.String("port", podPort))

	k.mux.Lock()
	requestID := k.requestID
	k.requestID++
	k.mux.Unlock()

	// create error stream
	headers := http.Header{}
	headers.Set(v1.StreamType, v1.StreamTypeError)
	headers.Set(v1.PortHeader, podPort)
	headers.Set(v1.PortForwardRequestIDHeader, strconv.Itoa(requestID))
	errorStream, err := connection.CreateStream(headers)
	if err != nil {
		k.logger.Error("error creating error stream", zap.Error(err), zap.String("pod", podName), zap.String("port", podPort))
		return err
	}
	// we're not writing to this stream
	errorStream.Close()
	defer connection.RemoveStreams(errorStream)

	errorChan := make(chan error)
	go func() {
		message, err := io.ReadAll(errorStream)
		switch {
		case err != nil:
			errorChan <- fmt.Errorf("error reading from error stream for %s:%s : %v", podName, podPort, err)
			k.logger.Error("error reading from error stream", zap.Error(err), zap.String("pod", podName), zap.String("port", podPort))
		case len(message) > 0:
			errorChan <- fmt.Errorf("error during forward request to kubeapi %s:%s : %s", podName, podPort, string(message))
			k.logger.Error("error during forwarding", zap.String("pod", podName), zap.String("port", podPort), zap.String("message", string(message)))
		}
		k.logger.Info("closing errorStream go routine")
		close(errorChan)
	}()

	// create data stream
	headers.Set(v1.StreamType, v1.StreamTypeData)
	dataStream, err := connection.CreateStream(headers)
	if err != nil {
		k.logger.Error("error creating forwarding stream", zap.Error(err), zap.String("pod", podName), zap.String("port", podPort))
		return err
	}
	defer connection.RemoveStreams(dataStream)

	streamRoutineDone := make(chan struct{})

	go func() {
		// Copy from the remote side to the local port.
		if _, err := io.Copy(channel, dataStream); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			k.logger.Error("error copying from remote stream to local connection", zap.Error(err))
		}
		// inform the select below that the remote copy is done
		streamRoutineDone <- struct{}{}
	}()

	go func() {
		// Copy from the local port to the remote side.
		if _, err := io.Copy(dataStream, channel); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			k.logger.Error("error copying from local connection to remote stream", zap.Error(err))
			// break out of the select below without waiting for the other copy to finish
		}
		// inform the select below that the local copy is done
		streamRoutineDone <- struct{}{}
	}()

	// wait for either a local->remote error or for copying from remote->local to finish
	select {
	case <-ctx.Done():
		if err := dataStream.Close(); err != nil {
			k.logger.Error("closing the dataStream", zap.Error(err))
		}
		return ctx.Err()
	case err := <-errorChan:
		if err := dataStream.Close(); err != nil {
			k.logger.Error("closing the dataStream", zap.Error(err))
		}
		return err
	case <-streamRoutineDone:
		k.logger.Info("stream routine done")
		return nil
	}
}
