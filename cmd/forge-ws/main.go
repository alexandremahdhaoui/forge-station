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

package main

import (
	"fmt"
	"os"

	"github.com/alexandremahdhaoui/forge-workspace/internal/adapter"
	"github.com/alexandremahdhaoui/forge-workspace/internal/controller/service"
	v1alpha1 "github.com/alexandremahdhaoui/forge-workspace/pkg/v1alpha1"
	"github.com/alexandremahdhaoui/forge/pkg/enginecli"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

func main() {
	svc, err := buildService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	enginecli.Bootstrap(enginecli.Config{
		Name:           "forge-ws",
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunCLI: func() error {
			return runCLI(svc, namespace)
		},
		RunMCP: func() error {
			return runMCP(svc, namespace)
		},
		FailureHandler: func(err error) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		},
	})
}

// buildService creates the WorkspaceService with K8s client, adapters, and service wiring.
func buildService() (service.WorkspaceService, error) {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(v1alpha1.AddToScheme(s))

	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig: %w", err)
	}

	c, err := client.New(config, client.Options{Scheme: s})
	if err != nil {
		return nil, fmt.Errorf("creating k8s client: %w", err)
	}

	wsAdapter := adapter.NewWorkspaceAdapter(c)
	secretAdapter := adapter.NewSecretAdapter(c)

	return service.NewWorkspaceService(wsAdapter, secretAdapter), nil
}
