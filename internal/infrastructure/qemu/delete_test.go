/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package qemu

import (
	"errors"
	"testing"

	"github.com/benschlueter/delegatio/internal/config/definitions"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestDeleteNetwork(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		connection    *stubLibvirtConnect
		expectErr     bool
		remainingNets map[string]bool
	}{
		"no nets": {
			connection: &stubLibvirtConnect{},
			expectErr:  false,
		},
		"other nets": {
			connection: &stubLibvirtConnect{
				networks: []*stubNetwork{
					{
						name:    "foo",
						freeErr: testErr,
					},
				},
			},
			remainingNets: map[string]bool{"foo": true},
			expectErr:     false,
		},
		"delegatio net": {
			connection: &stubLibvirtConnect{
				networks: []*stubNetwork{
					{
						name:    definitions.NetworkName,
						freeErr: testErr,
					},
				},
			},
			remainingNets: map[string]bool{definitions.NetworkName: false},
			expectErr:     false,
		},
		"delegatio net, destroy err": {
			connection: &stubLibvirtConnect{
				networks: []*stubNetwork{
					{
						name:       definitions.NetworkName,
						destroyErr: testErr,
					},
				},
			},
			remainingNets: map[string]bool{definitions.NetworkName: false},
			expectErr:     true,
		},
		"delegatio net, getName err": {
			connection: &stubLibvirtConnect{
				networks: []*stubNetwork{
					{
						name:       definitions.NetworkName,
						getNameErr: testErr,
					},
				},
			},
			remainingNets: map[string]bool{definitions.NetworkName: true},
			expectErr:     true,
		},
		"list all networks err": {
			connection: &stubLibvirtConnect{
				networks: []*stubNetwork{
					{
						name: definitions.NetworkName,
					},
				},
				listAllNetworksErr: testErr,
			},
			remainingNets: map[string]bool{definitions.NetworkName: true},
			expectErr:     true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)

			l := &LibvirtInstance{Conn: tc.connection}

			if tc.expectErr {
				assert.Error(l.deleteNetwork())
			} else {
				assert.NoError(l.deleteNetwork())
			}
			for key, v := range tc.remainingNets {
				assert.Equal(v, tc.connection.isNetworkPresent(key))
			}
		})
	}
}

func TestDeleteDomain(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		connection       *stubLibvirtConnect
		expectErr        bool
		remainingDomains map[string]bool
	}{
		"no domains": {
			connection: &stubLibvirtConnect{},
			expectErr:  false,
		},
		"other domains": {
			connection: &stubLibvirtConnect{
				domains: []*stubDomain{
					{
						name:    "foo",
						freeErr: testErr,
					},
				},
			},
			remainingDomains: map[string]bool{"foo": true},
			expectErr:        false,
		},
		"delegatio domain": {
			connection: &stubLibvirtConnect{
				domains: []*stubDomain{
					{
						name:    definitions.DomainPrefixMaster + "0",
						freeErr: testErr,
					},
				},
			},
			remainingDomains: map[string]bool{definitions.DomainPrefixMaster + "0": false},
			expectErr:        false,
		},
		"delegatio domain, destroy err": {
			connection: &stubLibvirtConnect{
				domains: []*stubDomain{
					{
						name:       definitions.DomainPrefixMaster + "0",
						destroyErr: testErr,
					},
				},
			},
			remainingDomains: map[string]bool{definitions.DomainPrefixMaster + "0": false},
			expectErr:        true,
		},
		"delegatio domain, getName err": {
			connection: &stubLibvirtConnect{
				domains: []*stubDomain{
					{
						name:       definitions.DomainPrefixMaster + "0",
						getNameErr: testErr,
					},
				},
			},
			remainingDomains: map[string]bool{definitions.DomainPrefixMaster + "0": true},
			expectErr:        true,
		},
		"list all domain error": {
			connection: &stubLibvirtConnect{
				domains: []*stubDomain{
					{
						name: definitions.DomainPrefixMaster + "0",
					},
				},
				listallDomainsErr: testErr,
			},
			remainingDomains: map[string]bool{definitions.DomainPrefixMaster + "0": true},
			expectErr:        true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)

			l := &LibvirtInstance{Conn: tc.connection}

			if tc.expectErr {
				assert.Error(l.deleteDomains())
			} else {
				assert.NoError(l.deleteDomains())
			}
			for key, v := range tc.remainingDomains {
				assert.Equal(v, tc.connection.isDomainPresent(key))
			}
		})
	}
}

func TestDeletePool(t *testing.T) {
	defer goleak.VerifyNone(t)
	testErr := errors.New("test error")
	testCases := map[string]struct {
		connection     *stubLibvirtConnect
		expectErr      bool
		remainingPools map[string]bool
	}{
		"no pool": {
			connection: &stubLibvirtConnect{
				pools: []*stubStoragePool{
					{
						name:    "foo",
						freeErr: testErr,
					},
				},
			},
			expectErr: false,
		},
		"delegatio pool active": {
			connection: &stubLibvirtConnect{
				pools: []*stubStoragePool{
					{
						name:    definitions.DiskPoolName,
						active:  true,
						freeErr: testErr,
					},
				},
			},
			remainingPools: map[string]bool{definitions.DiskPoolName: false},
			expectErr:      false,
		},
		"multiple pools active": {
			connection: &stubLibvirtConnect{
				pools: []*stubStoragePool{
					{
						name:    definitions.DiskPoolName,
						active:  true,
						freeErr: testErr,
					},
					{
						name:    "foo",
						active:  true,
						freeErr: testErr,
					},
				},
			},
			remainingPools: map[string]bool{definitions.DiskPoolName: false, "foo": true},
			expectErr:      false,
		},
		"get name error": {
			connection: &stubLibvirtConnect{
				pools: []*stubStoragePool{
					{
						name:       definitions.DiskPoolName,
						active:     true,
						getNameErr: testErr,
					},
				},
			},
			remainingPools: map[string]bool{definitions.DiskPoolName: true},
			expectErr:      true,
		},
		"is active error": {
			connection: &stubLibvirtConnect{
				pools: []*stubStoragePool{
					{
						name:        definitions.DiskPoolName,
						active:      true,
						isActiveErr: testErr,
					},
				},
			},
			remainingPools: map[string]bool{definitions.DiskPoolName: true},
			expectErr:      true,
		},
		"destroy error": {
			connection: &stubLibvirtConnect{
				pools: []*stubStoragePool{
					{
						name:       definitions.DiskPoolName,
						active:     true,
						destroyErr: testErr,
					},
				},
			},
			remainingPools: map[string]bool{definitions.DiskPoolName: false},
			expectErr:      true,
		},
		"delete error": {
			connection: &stubLibvirtConnect{
				pools: []*stubStoragePool{
					{
						name:      definitions.DiskPoolName,
						active:    true,
						deleteErr: testErr,
					},
				},
			},
			remainingPools: map[string]bool{definitions.DiskPoolName: false},
			expectErr:      true,
		},
		"undefine error": {
			connection: &stubLibvirtConnect{
				pools: []*stubStoragePool{
					{
						name:        definitions.DiskPoolName,
						active:      true,
						undefineErr: testErr,
					},
				},
			},
			remainingPools: map[string]bool{definitions.DiskPoolName: false},
			expectErr:      true,
		},
		"list all storage vol error": {
			connection: &stubLibvirtConnect{
				pools: []*stubStoragePool{
					{
						name:                 definitions.DiskPoolName,
						active:               true,
						listAllStorageVolErr: testErr,
					},
				},
			},
			remainingPools: map[string]bool{definitions.DiskPoolName: true},
			expectErr:      true,
		},
		"list all storage pool error": {
			connection: &stubLibvirtConnect{
				pools: []*stubStoragePool{
					{
						name:   definitions.DiskPoolName,
						active: true,
					},
				},
				listAllStoragePoolsErr: testErr,
			},
			remainingPools: map[string]bool{definitions.DiskPoolName: true},
			expectErr:      true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			// require := require.New(t)

			l := &LibvirtInstance{Conn: tc.connection}

			if tc.expectErr {
				assert.Error(l.deletePool())
			} else {
				assert.NoError(l.deletePool())
			}
			for poolName, v := range tc.remainingPools {
				assert.Equal(v, tc.connection.isPoolPresent(poolName))
			}
		})
	}
}
