//go:build e2e

// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"os"
	"testing"

	v1alpha1 "github.com/alexandremahdhaoui/forge-station/pkg/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	k8sClient  client.Client
	testScheme = runtime.NewScheme()
)

func TestMain(m *testing.M) {
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(v1alpha1.AddToScheme(testScheme))

	// KUBECONFIG is set by testenv-kind
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		panic("KUBECONFIG not set — e2e tests require a Kind cluster")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic("failed to build kubeconfig: " + err.Error())
	}

	k8sClient, err = client.New(config, client.Options{Scheme: testScheme})
	if err != nil {
		panic("failed to create k8s client: " + err.Error())
	}

	os.Exit(m.Run())
}
