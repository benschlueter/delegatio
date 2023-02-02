/* SPDX-License-Identifier: AGPL-3.0-only
* Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
)

func TestUploadBaseImage(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		connection *stubLibvirtConnect
		stVolume   *stubVolume
		expectErr  bool
		activePool bool
		cancelCtx  bool
		noFile     bool
	}{
		"create base img": {
			connection: &stubLibvirtConnect{},
			stVolume:   &stubVolume{},
			expectErr:  false,
		},
		"cancelled context ": {
			connection: &stubLibvirtConnect{},
			stVolume:   &stubVolume{},
			expectErr:  true,
			cancelCtx:  true,
		},
		"new stream err ": {
			connection: &stubLibvirtConnect{
				newStreamErr: testErr,
			},
			stVolume:  &stubVolume{},
			expectErr: true,
		},
		"upload err ": {
			connection: &stubLibvirtConnect{},
			stVolume: &stubVolume{
				uploadErr: testErr,
			},
			expectErr: true,
		},
		"open err ": {
			connection: &stubLibvirtConnect{},
			stVolume:   &stubVolume{},
			expectErr:  true,
			noFile:     true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Must fix the stream stuff before this can be tested
			assert := assert.New(t)
			require := require.New(t)

			imagePath := "test.img"
			fs := afero.NewMemMapFs()
			if !tc.noFile {
				file, err := fs.Create(imagePath)
				require.NoError(err)
				_, err = file.Write([]byte("test"))
				require.NoError(err)
				require.NoError(file.Close())
			}

			l := &LibvirtInstance{Conn: tc.connection, Log: zap.NewNop(), ImagePath: imagePath, fs: &afero.Afero{Fs: fs}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancelCtx {
				cancel()
			}

			if tc.expectErr {
				assert.Error(l.uploadBaseImage(ctx, tc.stVolume))
			} else {
				assert.NoError(l.uploadBaseImage(ctx, tc.stVolume))
			}
		})
	}
}

func TestDeleteVolumesFromPool(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		pool       *stubStoragePool
		expectErr  bool
		activePool bool
	}{
		"normal": {
			pool: &stubStoragePool{
				volumes: []*stubVolume{
					{
						freeErr: testErr,
					},
				},
			},
			expectErr: false,
		},
		"list volumes error ": {
			pool: &stubStoragePool{
				listAllStorageVolErr: testErr,
			},
			expectErr: true,
		},
		"delete volume error ": {
			pool: &stubStoragePool{
				volumes: []*stubVolume{
					{
						deleteErr: testErr,
					},
				},
			},
			expectErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Must fix the stream stuff before this can be tested
			assert := assert.New(t)
			// require := require.New(t)

			if tc.expectErr {
				assert.Error(deleteVolumesFromPool(tc.pool))
			} else {
				assert.NoError(deleteVolumesFromPool(tc.pool))
			}
		})
	}
}

func TestGetControlPlaneIP(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		connection *stubLibvirtConnect
		expectErr  bool
		activePool bool
	}{
		"normal": {
			connection: &stubLibvirtConnect{
				domains: []*stubDomain{
					{
						name: "delegatio-master-0",
						netIfaces: []libvirt.DomainInterface{
							{
								Name: "lo",
							},
							{
								Name: "eth0",
								Addrs: []libvirt.DomainIPAddress{
									{
										Type: libvirt.IP_ADDR_TYPE_IPV4,
										Addr: "0.0.1.0",
									},
								},
							},
						},
					},
				},
			},
		},
		"no ip addr": {
			connection: &stubLibvirtConnect{
				domains: []*stubDomain{
					{
						name: "delegatio-master-0",
						netIfaces: []libvirt.DomainInterface{
							{
								Name: "eth0",
								Addrs: []libvirt.DomainIPAddress{
									{
										Type: libvirt.IP_ADDR_TYPE_IPV4,
									},
								},
							},
						},
					},
				},
			},
			expectErr: true,
		},
		"domain lookup err": {
			connection: &stubLibvirtConnect{
				lookUpDomainErr: testErr,
			},
			expectErr: true,
		},
		"list all interface addrs err": {
			connection: &stubLibvirtConnect{
				domains: []*stubDomain{
					{
						name:                         "delegatio-master-0",
						listAllInterfaceAddressesErr: testErr,
					},
				},
			},
			expectErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Must fix the stream stuff before this can be tested
			assert := assert.New(t)
			// require := require.New(t)
			l := &LibvirtInstance{Conn: tc.connection, Log: zap.NewNop()}

			_, err := l.getControlPlaneIP()
			if tc.expectErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
