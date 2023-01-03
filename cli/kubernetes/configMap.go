package kubernetes

import (
	"context"

	"go.uber.org/zap"
	coreAPI "k8s.io/api/core/v1"
	metaAPI "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KubernetesClient) CreateConfigMap(ctx context.Context, name, namespace string) error {
	cm := coreAPI.ConfigMap{
		TypeMeta: metaAPI.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metaAPI.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	_, err := k.client.CoreV1().ConfigMaps(namespace).Create(ctx, &cm, metaAPI.CreateOptions{})
	return err
}

func (k *KubernetesClient) AddDataToConfigMap(ctx context.Context, mapName, namespace, key, value string) error {
	cfgMap, err := k.client.CoreV1().ConfigMaps(namespace).Get(ctx, mapName, metaAPI.GetOptions{})
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
	_, err = k.client.CoreV1().ConfigMaps(namespace).Update(ctx, cfgMap, metaAPI.UpdateOptions{})
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
