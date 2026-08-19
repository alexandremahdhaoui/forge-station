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
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexandremahdhaoui/forge-station/internal/adapter"
	"github.com/alexandremahdhaoui/forge-station/internal/controller/service"
	restdriver "github.com/alexandremahdhaoui/forge-station/internal/driver/rest"
	v1alpha1 "github.com/alexandremahdhaoui/forge-station/pkg/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	// Scheme.
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(v1alpha1.AddToScheme(s))

	// K8s client.
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("building kubeconfig: %v", err)
	}

	c, err := client.New(config, client.Options{Scheme: s})
	if err != nil {
		log.Fatalf("creating k8s client: %v", err)
	}

	// Layer 1: Adapters.
	wsAdapter := adapter.NewWorkspaceAdapter(c)
	secretAdapter := adapter.NewSecretAdapter(c)
	portfolioAdapter := adapter.NewPortfolioAdapter(c)

	// Layer 2: Services.
	wsSvc := service.NewWorkspaceService(wsAdapter, secretAdapter)
	pSvc := service.NewPortfolioService(portfolioAdapter)

	// Layer 3: REST handler.
	handler := restdriver.NewAPIHandler(wsSvc, pSvc)

	// Layer 4: HTTP server.
	mux := http.NewServeMux()
	si := restdriver.NewStrictHandler(handler, nil)
	restdriver.HandlerFromMux(si, mux)

	// Health and readiness checks.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Printf("forge-ws-api listening on %s", addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down...")
	_ = srv.Close()
}
