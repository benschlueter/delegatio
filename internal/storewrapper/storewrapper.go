/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package storewrapper

import (
	"encoding/json"

	"github.com/benschlueter/delegatio/internal/store"
)

const (
	challengeLocationPrefix = "challenge-"
	publicKeyPrefix         = "peersResourceVersion"
)

// StoreWrapper is a wrapper for the store interface.
type StoreWrapper struct {
	Store interface {
		Get(string) ([]byte, error)
		Put(string, []byte) error
		Delete(string) error
		Iterator(string) (store.Iterator, error)
	}
}

// PutChallenge puts a challenge into the store.
func (s StoreWrapper) PutChallenge(challengeName string, target any) error {
	challengeData, err := json.Marshal(target)
	if err != nil {
		return err
	}
	return s.Store.Put(challengeLocationPrefix+challengeName, challengeData)
}

// GetChallenge gets a challenge.
func (s StoreWrapper) GetChallenge(challengeName string, target any) error {
	challengeData, err := s.Store.Get(challengeLocationPrefix + challengeName)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(challengeData, target); err != nil {
		return err
	}
	return nil
}

// PutPublicKey puts a publicKey and associated data of the key into the store.
func (s StoreWrapper) PutPublicKey(pubkey string, target any) error {
	publicKeyData, err := json.Marshal(target)
	if err != nil {
		return err
	}
	return s.Store.Put(publicKeyPrefix+pubkey, publicKeyData)
}

// GetPublicKey gets data associated with the publicKey.
func (s StoreWrapper) GetPublicKey(publickey string, target any) error {
	publicKeyData, err := s.Store.Get(publicKeyPrefix + publickey)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(publicKeyData, target); err != nil {
		return err
	}
	return nil
}

// GetAllChallenges gets all challenge names.
func (s StoreWrapper) GetAllChallenges(publickey string, target any) (challenges []string, err error) {
	chIterator, err := s.Store.Iterator(challengeLocationPrefix)
	if err != nil {
		return
	}
	for chIterator.HasNext() {
		challenge, err := chIterator.GetNext()
		if err != nil {
			return nil, err
		}
		challenges = append(challenges, challenge)
	}
	return
}
