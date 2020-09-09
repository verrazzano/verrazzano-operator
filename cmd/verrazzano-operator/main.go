// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/verrazzano/verrazzano-operator/pkg/api/applications"
	"github.com/verrazzano/verrazzano-operator/pkg/api/clusters"
	"github.com/verrazzano/verrazzano-operator/pkg/api/domains"
	"github.com/verrazzano/verrazzano-operator/pkg/api/grids"
	"github.com/verrazzano/verrazzano-operator/pkg/api/images"
	"github.com/verrazzano/verrazzano-operator/pkg/api/instance"
	"github.com/verrazzano/verrazzano-operator/pkg/api/jobs"
	"github.com/verrazzano/verrazzano-operator/pkg/api/microservices"
	"github.com/verrazzano/verrazzano-operator/pkg/api/operators"
	"github.com/verrazzano/verrazzano-operator/pkg/api/secrets"
	pkgverrazzanooperator "github.com/verrazzano/verrazzano-operator/pkg/controller"
	"github.com/verrazzano/verrazzano-operator/pkg/util"
	"github.com/verrazzano/verrazzano-operator/pkg/util/apiserver"
	"github.com/verrazzano/verrazzano-operator/pkg/util/logs"
)

var (
	masterURL                    string
	kubeconfig                   string
	watchNamespace               string
	verrazzanoUri                string
	helidonAppOperatorDeployment string
	enableMonitoringStorage      string
	apiServerRealm               string
)

// homePage provides a default handler for unmatched patterns
func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "verrazzano-operator apiserver")
	fmt.Println("GET /")
}

// handleRequests registers the routes and handlers
func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)

	myRouter.HandleFunc("/", homePage)

	// There is only 1
	myRouter.Path("/instance").Methods("GET").HandlerFunc(instance.ReturnSingleInstance)

	// Routes are tested in the order they were added to the router. If two routes match, the first one wins
	myRouter.Path("/applications").Methods("GET").HandlerFunc(applications.ReturnAllApplications)
	myRouter.Path("/applications").Methods("POST").HandlerFunc(applications.CreateNewApplication)
	myRouter.Path("/applications/{id}").Methods("GET").HandlerFunc(applications.ReturnSingleApplication)
	myRouter.Path("/applications/{id}").Methods("DELETE").HandlerFunc(applications.DeleteApplication)
	myRouter.Path("/applications/{id}").Methods("PUT").HandlerFunc(applications.UpdateApplication)

	myRouter.Path("/clusters").Methods("GET").HandlerFunc(clusters.ReturnAllClusters)
	myRouter.Path("/clusters/{id}").Methods("GET").HandlerFunc(clusters.ReturnSingleCluster)

	myRouter.Path("/domains").Methods("GET").HandlerFunc(domains.ReturnAllDomains)
	myRouter.Path("/domains/{id}").Methods("GET").HandlerFunc(domains.ReturnSingleDomain)

	myRouter.Path("/grids").Methods("GET").HandlerFunc(grids.ReturnAllGrids)
	myRouter.Path("/grids/{id}").Methods("GET").HandlerFunc(grids.ReturnSingleGrid)

	myRouter.Path("/images").Methods("GET").HandlerFunc(images.ReturnAllImages)
	myRouter.Path("/images/{id}").Methods("GET").HandlerFunc(images.ReturnSingleImage)

	myRouter.Path("/jobs").Methods("GET").HandlerFunc(jobs.ReturnAllJobs)
	myRouter.Path("/jobs/{id}").Methods("GET").HandlerFunc(jobs.ReturnSingleJob)

	myRouter.Path("/microservices").Methods("GET").HandlerFunc(microservices.ReturnAllMicroservices)
	myRouter.Path("/microservices/{id}").Methods("GET").HandlerFunc(microservices.ReturnSingleMicroservice)

	myRouter.Path("/operators").Methods("GET").HandlerFunc(operators.ReturnAllOperators)
	myRouter.Path("/operators/{id}").Methods("GET").HandlerFunc(operators.ReturnSingleOperator)

	myRouter.Path("/secrets").Methods("GET").HandlerFunc(secrets.ReturnAllSecrets)
	myRouter.Path("/secrets").Methods("POST").HandlerFunc(secrets.CreateSecret)
	myRouter.Path("/secrets/{id}").Methods("GET").HandlerFunc(secrets.ReturnSingleSecret)
	myRouter.Path("/secrets/{id}").Methods("DELETE").HandlerFunc(secrets.DeleteSecret)
	myRouter.Path("/secrets/{id}").Methods("PATCH").HandlerFunc(secrets.UpdateSecret)

	glog.Info("Starting API server...")

	if apiServerRealm == "" {
		glog.Warning("Bearer token verification is disabled as apiServerRealm is not specified")
	}
	glog.Fatal(http.ListenAndServe(":3456", apiserver.CORSHandler(apiserver.AuthHandler(myRouter))))
}

func main() {

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

	// Load Verrazzano Operator Manifest

	manifest, err := util.LoadManifest()
	if err != nil {
		glog.Fatalf("Error loading manifest: %s", err.Error())
	}

	flag.Parse()

	logs.InitLogs()

	glog.V(6).Infof("Creating new controller watching namespace %s.", watchNamespace)
	if strings.EqualFold(helidonAppOperatorDeployment, "false") {
		glog.V(6).Info("helidonAppOperatorDeployment Disabled")
		manifest.HelidonAppOperatorImage = ""
	}
	controller, err := pkgverrazzanooperator.NewController(kubeconfig, manifest, masterURL, watchNamespace, verrazzanoUri, enableMonitoringStorage)
	if err != nil {
		glog.Fatalf("Error creating the controller: %s", err.Error())
	}

	//
	// start the REST API Server (for web ui)
	//

	// give the handlers access to the informers
	applications.Init(controller.ListerSet())
	clusters.Init(controller.ListerSet())
	domains.Init(controller.ListerSet())
	grids.Init(controller.ListerSet())
	images.Init()
	jobs.Init()
	microservices.Init(controller.ListerSet())
	operators.Init(controller.ListerSet())
	secrets.Init(controller.ListerSet())
	apiserver.Init(controller.ListerSet())

	instance.SetVerrazzanoUri(verrazzanoUri)
	apiserver.SetRealm(apiServerRealm)
	// start REST handlers
	go handleRequests()

	//
	// start the controller
	//

	if err = controller.Run(2); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}

}

func init() {
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&watchNamespace, "watchNamespace", "", "Optionally, a namespace to watch exclusively.  If not set, all namespaces will be watched.")
	flag.StringVar(&verrazzanoUri, "verrazzanoUri", "", "Verrazzano URI, for example my-verrazzano-1.verrazzano.example.com")
	flag.StringVar(&helidonAppOperatorDeployment, "helidonAppOperatorDeployment", "", "--helidonAppOperatorDeployment=false disables helidonAppOperatorDeployment")
	flag.StringVar(&enableMonitoringStorage, "enableMonitoringStorage", "", "Enable storage for monitoring.  The default is true. 'false' means monitoring storage is disabled.")
	flag.StringVar(&apiServerRealm, "apiServerRealm", "", "API Server Realm on Keycloak")
}
