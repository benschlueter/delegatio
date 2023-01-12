/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package helpers

import (
	"context"

	"go.uber.org/zap"
	coreAPI "k8s.io/api/core/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateConfigMap creates a configmap.
func (k *Client) CreateConfigMap(ctx context.Context, namespace, name string) error {
	cm := coreAPI.ConfigMap{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: coreAPI.SchemeGroupVersion.Version,
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	_, err := k.Client.CoreV1().ConfigMaps(namespace).Create(ctx, &cm, metaAPI.CreateOptions{})
	return err
}

// AddDataToConfigMap adds data do a given configmap.
func (k *Client) AddDataToConfigMap(ctx context.Context, namespace, mapName, key, value string) error {
	cfgMap, err := k.Client.CoreV1().ConfigMaps(namespace).Get(ctx, mapName, metaAPI.GetOptions{})
	if err != nil {
		return err
	}
	if len(cfgMap.Data) == 0 {
		cfgMap.Data = map[string]string{
			key: value,
		}
	} else {
		cfgMap.Data[key] = value
	}
	_, err = k.Client.CoreV1().ConfigMaps(namespace).Update(ctx, cfgMap, metaAPI.UpdateOptions{})
	if err != nil {
		k.logger.Error("updating configMap",
			zap.Error(err),
			zap.String("mapname", mapName),
			zap.String("namespace", namespace),
			zap.String("key", key),
			zap.String("value", value))
		return err
	}
	k.logger.Debug("updating configMap",
		zap.Error(err),
		zap.String("mapname", mapName),
		zap.String("namespace", namespace),
		zap.String("key", key),
		zap.String("value", value))
	return nil
}

// GetConfigMapData gets data from a given configmap.
func (k *Client) GetConfigMapData(ctx context.Context, namespace, mapName string) (map[string]string, error) {
	cfgMap, err := k.Client.CoreV1().ConfigMaps(namespace).Get(ctx, mapName, metaAPI.GetOptions{})
	if err != nil {
		return nil, err
	}
	return cfgMap.Data, nil
}
