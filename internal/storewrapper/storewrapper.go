/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package storewrapper

import (
	"encoding/json"
	"errors"

	"github.com/benschlueter/delegatio/internal/store"
)

const (
	challengeLocationPrefix = "challenge-"
	publicKeyPrefix         = "publickey-"
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

// PutChallengeData puts a challenge into the store.
func (s StoreWrapper) PutChallengeData(challengeName string, target any) error {
	challengeData, err := json.Marshal(target)
	if err != nil {
		return err
	}
	return s.Store.Put(challengeLocationPrefix+challengeName, challengeData)
}

// GetChallengeData gets a challenge.
func (s StoreWrapper) GetChallengeData(challengeName string, target any) error {
	challengeData, err := s.Store.Get(challengeLocationPrefix + challengeName)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(challengeData, target); err != nil {
		return err
	}
	return nil
}

// ChallengeExists checks whether the challenge is in the store.
func (s StoreWrapper) ChallengeExists(challengeName string) (bool, error) {
	_, err := s.Store.Get(challengeLocationPrefix + challengeName)
	if errors.Is(err, &store.ValueUnsetError{}) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// PutPublicKeyData puts a publicKey and associated data of the key into the store.
func (s StoreWrapper) PutPublicKeyData(pubkey string, target any) error {
	publicKeyData, err := json.Marshal(target)
	if err != nil {
		return err
	}
	return s.Store.Put(publicKeyPrefix+pubkey, publicKeyData)
}

// GetPublicKeyData gets data associated with the publicKey.
func (s StoreWrapper) GetPublicKeyData(publickey string, target any) error {
	publicKeyData, err := s.Store.Get(publicKeyPrefix + publickey)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(publicKeyData, target); err != nil {
		return err
	}
	return nil
}

// PublicKeyExists checks whether the publicKey is in the store.
func (s StoreWrapper) PublicKeyExists(publicKey string) (bool, error) {
	_, err := s.Store.Get(publicKeyPrefix + publicKey)
	if errors.Is(err, &store.ValueUnsetError{}) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetAllChallenges gets all challenge names.
func (s StoreWrapper) GetAllChallenges(publickey string) (challenges []string, err error) {
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

// GetAllKeys prints everything in the store.
func (s StoreWrapper) GetAllKeys() (keys []string, err error) {
	stIterator, err := s.Store.Iterator("")
	if err != nil {
		return
	}
	for stIterator.HasNext() {
		key, err := stIterator.GetNext()
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return
}
