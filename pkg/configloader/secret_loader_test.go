/*
 * Copyright 2022 The Furiko Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package configloader_test

import (
	"context"
	"encoding/base64"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakeclientset "k8s.io/client-go/kubernetes/fake"

	configv1 "github.com/furiko-io/furiko/apis/config/v1"
	"github.com/furiko-io/furiko/pkg/configloader"
)

const (
	secretNamespace = "test-namespace"
	secretName      = "test-secret"
)

type secretLoaderTest struct{}

func (c *secretLoaderTest) Create(ctx context.Context, client kubernetes.Interface) error {
	_, err := client.CoreV1().Secrets(secretNamespace).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: secretNamespace,
			Name:      secretName,
		},
	}, metav1.CreateOptions{})
	return err
}

func (c *secretLoaderTest) Update(ctx context.Context, client kubernetes.Interface, data string) error {
	_, err := client.CoreV1().Secrets(secretNamespace).Update(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: secretNamespace,
			Name:      secretName,
		},
		Data: map[string][]byte{
			string(configv1.ConfigNameJobController): []byte(base64.StdEncoding.EncodeToString([]byte(data))),
		},
	}, metav1.UpdateOptions{})
	return err
}

func TestSecretLoader(t *testing.T) {
	client := fakeclientset.NewSimpleClientset()
	loader := configloader.NewSecretLoader(client, secretNamespace, secretName)
	testKubernetesLoader(t, client, loader, &secretLoaderTest{})
}
