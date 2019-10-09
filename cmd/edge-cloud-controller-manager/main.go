/*
Copyright 2019 Midokura

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// The external controller manager is responsible for running controller loops that
// are cloud provider dependent. It uses the API to listen to new events on resources.

package main

import (
	goflag "flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/midokura/cloud-provider-edge/pkg/cloudprovider/providers/edge"
	_ "github.com/spf13/cobra"
	"github.com/spf13/pflag"

	_ "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/component-base/cli/flag"
	"k8s.io/klog"
	"k8s.io/kubernetes/cmd/cloud-controller-manager/app"
	"k8s.io/kubernetes/cmd/cloud-controller-manager/app/options"
	_ "k8s.io/kubernetes/pkg/client/metrics/prometheus" // for client metric registration
	_ "k8s.io/kubernetes/pkg/features"                  // add the kubernetes feature gates
	_ "k8s.io/kubernetes/pkg/util/flag"
	_ "k8s.io/kubernetes/pkg/version/prometheus" // for version metric registration
	_ "k8s.io/kubernetes/pkg/version/verflag"
)

// Version is set by the linker flags in the Makefile.
var version string

func init() {
	mux := http.NewServeMux()
	healthz.InstallHandler(mux)
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	goflag.CommandLine.Parse([]string{})
	s, err := options.NewCloudControllerManagerOptions()
	if err != nil {
		klog.Fatalf("unable to initialize command options: %v", err)
	}

	command := app.NewCloudControllerManagerCommand()

	fs := command.Flags()
	namedFlagSets := s.Flags(nil, nil)
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	pflag.CommandLine.SetNormalizeFunc(flag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	defer klog.Flush()

	klog.Infof("edge-cloud-controller-manager version: %s", version)

	s.KubeCloudShared.CloudProvider.Name = edge.ProviderName
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
