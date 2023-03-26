/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package storewrapper

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/benschlueter/delegatio/internal/config"
	"github.com/benschlueter/delegatio/internal/store"
)

const (
	challengeLocationPrefix = "challenge-"
	publicKeyPrefix         = "publickey-"
	privKeyLocation         = "privkey-ssh"
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
	return json.Unmarshal(challengeData, target)
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

// GetAllChallenges gets all challenge names.
func (s StoreWrapper) GetAllChallenges() (map[string]config.ChallengeInformation, error) {
	chIterator, err := s.Store.Iterator(challengeLocationPrefix)
	if err != nil {
		return nil, err
	}
	challenges := make(map[string]config.ChallengeInformation)
	for chIterator.HasNext() {
		key, err := chIterator.GetNext()
		if err != nil {
			return nil, err
		}
		key = strings.TrimPrefix(key, challengeLocationPrefix)
		var challenge config.ChallengeInformation
		if err := s.GetChallengeData(key, &challenge); err != nil {
			return nil, err
		}
		challenges[key] = challenge
	}
	return challenges, nil
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
	return json.Unmarshal(publicKeyData, target)
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

// GetAllPublicKeys gets all publicKeys and the associated user information.
func (s StoreWrapper) GetAllPublicKeys() (map[string]config.UserInformation, error) {
	pubKeyIterator, err := s.Store.Iterator(publicKeyPrefix)
	if err != nil {
		return nil, err
	}
	userData := make(map[string]config.UserInformation)
	for pubKeyIterator.HasNext() {
		key, err := pubKeyIterator.GetNext()
		if err != nil {
			return nil, err
		}
		key = strings.TrimPrefix(key, publicKeyPrefix)
		var user config.UserInformation
		if err := s.GetPublicKeyData(key, &user); err != nil {
			return nil, err
		}
		userData[key] = user
	}
	return userData, nil
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

// PutPrivKey puts a privKey  into the store.
func (s StoreWrapper) PutPrivKey(privkey []byte) error {
	return s.Store.Put(privKeyLocation, privkey)
}

// GetPrivKey gets the privKey.
func (s StoreWrapper) GetPrivKey() ([]byte, error) {
	return s.Store.Get(privKeyLocation)
}
