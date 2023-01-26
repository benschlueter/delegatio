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
)

func TestCreateStoragePool(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		connection *stubLibvirtConnect
		expectErr  bool
		activePool bool
	}{
		"delegatio pool": {
			connection: &stubLibvirtConnect{
				storagePoolDefine: &stubStoragePool{},
			},
			activePool: true,
			expectErr:  false,
		},
		"build error": {
			connection: &stubLibvirtConnect{
				storagePoolDefine: &stubStoragePool{
					buildErr: testErr,
				},
			},
			expectErr: true,
		},
		"create error": {
			connection: &stubLibvirtConnect{
				storagePoolDefine: &stubStoragePool{
					createErr: testErr,
				},
			},
			expectErr: true,
		},
		"storage pool define err": {
			connection: &stubLibvirtConnect{
				storagePoolDefine:    &stubStoragePool{},
				storagePoolDefineErr: testErr,
			},
			expectErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)

			l := &libvirtInstance{Conn: tc.connection, Log: zap.NewNop()}

			if tc.expectErr {
				assert.Error(l.createStoragePool())
			} else {
				assert.NoError(l.createStoragePool())
			}
			if tc.activePool {
				assert.True(tc.connection.storagePoolDefine.active)
			}
		})
	}
}

func TestCreateBaseImage(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		connection *stubLibvirtConnect
		expectErr  bool
		activePool bool
	}{
		"create base img": {
			connection: &stubLibvirtConnect{
				storagePoolTargetPath: &stubStoragePool{
					StorageVolCreate: &stubVolume{},
				},
			},
			expectErr: false,
		},
		"lookup pool err": {
			connection: &stubLibvirtConnect{
				storagePoolTargetPath: &stubStoragePool{
					StorageVolCreate: &stubVolume{},
				},
				storagePoolTargetPathErr: testErr,
			},
			expectErr: true,
		},
		"vol create err": {
			connection: &stubLibvirtConnect{
				storagePoolTargetPath: &stubStoragePool{
					StorageVolCreate:    &stubVolume{},
					StorageVolCreateErr: testErr,
				},
			},
			expectErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Must fix the stream stuff before this can be tested
			assert := assert.New(t)
			require := require.New(t)

			imagePath := "test.img"
			fs := afero.NewMemMapFs()
			file, err := fs.Create(imagePath)
			require.NoError(err)
			l := &libvirtInstance{Conn: tc.connection, Log: zap.NewNop(), ImagePath: imagePath, fs: &afero.Afero{Fs: fs}}
			_, err = file.Write([]byte("test"))
			require.NoError(err)
			require.NoError(file.Close())

			if tc.expectErr {
				assert.Error(l.createBaseImage(context.Background()))
			} else {
				assert.NoError(l.createBaseImage(context.Background()))
			}
		})
	}
}

func TestCreateBootImage(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		connection *stubLibvirtConnect
		expectErr  bool
		activePool bool
		id         string
	}{
		"create": {
			connection: &stubLibvirtConnect{
				storagePoolTargetPath: &stubStoragePool{
					StorageVolCreate: &stubVolume{},
				},
			},
			id:        "test",
			expectErr: false,
		},
		"lookup error ": {
			connection: &stubLibvirtConnect{
				storagePoolTargetPath: &stubStoragePool{
					StorageVolCreate: &stubVolume{},
				},
				storagePoolTargetPathErr: testErr,
			},
			id:        "test",
			expectErr: true,
		},
		"volume create error ": {
			connection: &stubLibvirtConnect{
				storagePoolTargetPath: &stubStoragePool{
					StorageVolCreate:    &stubVolume{},
					StorageVolCreateErr: testErr,
				},
			},
			id:        "test",
			expectErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)

			l := &libvirtInstance{Conn: tc.connection, Log: zap.NewNop()}

			if tc.expectErr {
				assert.Error(l.createBootImage(tc.id))
			} else {
				assert.NoError(l.createBootImage(tc.id))
			}
		})
	}
}

func TestCreateNetwork(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		connection *stubLibvirtConnect
		expectErr  bool
	}{
		"create": {
			connection: &stubLibvirtConnect{
				networkCreate: &stubNetwork{},
			},
			expectErr: false,
		},
		"create error": {
			connection: &stubLibvirtConnect{
				networkCreate:    &stubNetwork{},
				networkCreateErr: testErr,
			},
			expectErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)

			l := &libvirtInstance{Conn: tc.connection, Log: zap.NewNop()}

			if tc.expectErr {
				assert.Error(l.createNetwork())
			} else {
				assert.NoError(l.createNetwork())
			}
		})
	}
}

func TestCreateDomain(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		connection *stubLibvirtConnect
		expectErr  bool
	}{
		"create": {
			connection: &stubLibvirtConnect{
				domainCreate: &stubDomain{},
			},
			expectErr: false,
		},
		"create error": {
			connection: &stubLibvirtConnect{
				domainCreate:    &stubDomain{},
				domainCreateErr: testErr,
			},
			expectErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)

			l := &libvirtInstance{Conn: tc.connection, Log: zap.NewNop()}

			if tc.expectErr {
				assert.Error(l.createDomain("test"))
			} else {
				assert.NoError(l.createDomain("test"))
			}
		})
	}
}

func TestCreateInstance(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		connection *stubLibvirtConnect
		expectErr  bool
		masterNode bool
	}{
		"create": {
			connection: &stubLibvirtConnect{
				domainCreate: &stubDomain{},
				storagePoolTargetPath: &stubStoragePool{
					StorageVolCreate: &stubVolume{},
				},
			},
			expectErr: false,
		},
		"boot img error": {
			connection: &stubLibvirtConnect{
				domainCreate: &stubDomain{},
				storagePoolTargetPath: &stubStoragePool{
					StorageVolCreate:    &stubVolume{},
					StorageVolCreateErr: testErr,
				},
			},
			expectErr: true,
		},
		"domain error": {
			connection: &stubLibvirtConnect{
				domainCreate:    &stubDomain{},
				domainCreateErr: testErr,
				storagePoolTargetPath: &stubStoragePool{
					StorageVolCreate: &stubVolume{},
				},
			},
			expectErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)

			l := &libvirtInstance{Conn: tc.connection, Log: zap.NewNop()}

			if tc.expectErr {
				assert.Error(l.CreateInstance("test", tc.masterNode))
			} else {
				assert.NoError(l.CreateInstance("test", tc.masterNode))
			}
		})
	}
}
