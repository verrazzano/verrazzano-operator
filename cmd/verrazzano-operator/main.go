// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"

	pkgverrazzanooperator "github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-operator/pkg/util/logs"
	"go.uber.org/zap"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	masterURL               string
	kubeconfig              string
	watchNamespace          string
	verrazzanoURI           string
	obsoleteFlag            string
	enableMonitoringStorage string
	apiServerRealm          string
	startController         bool
	zapOptions              = kzap.Options{}
)

const apiVersionPrefix = "/20210501"

func prepare() error {
	flag.Parse()
	fmt.Println(" _    _                                                                    _____")
	fmt.Println("| |  | |                                                                  / ___ \\                              _")
	fmt.Println("| |  | |  ____   ____   ____   ____  _____  _____   ____  ____    ___    | |   | | ____    ____   ____   ____ | |_    ___    ____")
	fmt.Println(" \\ \\/ /  / _  ) / ___) / ___) / _  |(___  )(___  ) / _  ||  _ \\  / _ \\   | |   | ||  _ \\  / _  ) / ___) / _  ||  _)  / _ \\  / ___)")
	fmt.Println("  \\  /  ( (/ / | |    | |    ( ( | | / __/  / __/ ( ( | || | | || |_| |  | |___| || | | |( (/ / | |    ( ( | || |__ | |_| || |")
	fmt.Println("   \\/    \\____)|_|    |_|     \\_||_|(_____)(_____) \\_||_||_| |_| \\___/    \\_____/ | ||_/  \\____)|_|     \\_||_| \\___) \\___/ |_|")
	fmt.Println("                                                                                  |_|")
	fmt.Println("")
	fmt.Println("          Verrazzano Operator")
	fmt.Println("")
	logs.InitLogs(zapOptions)
	return nil
}

func main() {

	err := prepare()
	if err != nil {
		zap.S().Fatalf("Failed loading manifest: %v")
	}
	zap.S().Infof("Creating new controller watching namespace %s.", watchNamespace)

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		zap.S().Fatalf("Failed creating the controller configuration: %v", err)
	}
	controller, err := pkgverrazzanooperator.NewController(config, watchNamespace, verrazzanoURI, enableMonitoringStorage)
	if err != nil {
		zap.S().Fatalf("Failed creating the controller: %v", err)
	}

	apiServerExited := make(chan bool)

	if startController {
		// start the controller
		if err = controller.Run(2, kubeconfig); err != nil {
			zap.S().Fatalf("Failed running controller: %v", err)
		}
	}

	zap.S().Infow("Waiting for api server to exit")
	<-apiServerExited
}

func init() {
	initFlags(flag.StringVar, flag.BoolVar, zapOptions.BindFlags)
}

// initFlags registers all command line flags. It accepts a registration function for string vars and
// one for boolean vars, to enable testing
func initFlags(
	stringVarFunc func(p *string, name string, value string, usage string),
	boolVarFunc func(p *bool, name string, value bool, usage string),
	flagSetFunc func(fs *flag.FlagSet)) {
	stringVarFunc(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	stringVarFunc(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	stringVarFunc(&watchNamespace, "watchNamespace", "", "Optionally, a namespace to watch exclusively.  If not set, all namespaces will be watched.")
	stringVarFunc(&verrazzanoURI, "verrazzanoUri", "", "Verrazzano URI, for example my-verrazzano-1.verrazzano.example.com")
	stringVarFunc(&obsoleteFlag, "helidonAppOperatorDeployment", "", "--helidonAppOperatorDeployment=false disables helidonAppOperatorDeployment")
	stringVarFunc(&enableMonitoringStorage, "enableMonitoringStorage", "true", "Enable storage for monitoring.  The default is true. 'false' means monitoring storage is disabled.")
	stringVarFunc(&apiServerRealm, "apiServerRealm", "", "API Server Realm on Keycloak")
	boolVarFunc(&startController, "startController", true, "Whether to start the Kubernetes controller (true by default)")
	flagSetFunc(flag.CommandLine)
}
