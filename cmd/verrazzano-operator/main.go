// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"fmt"
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"strings"

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
	verrazzanoURI                string
	helidonAppOperatorDeployment string
	enableMonitoringStorage      string
	apiServerRealm               string
	startController              bool
)

const apiVersionPrefix = "/20210501"

// homePage provides a default handler for unmatched patterns
func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "verrazzano-operator apiserver")
	fmt.Println("GET /")
}

func handleRequests(apiServerFinished chan bool, rootRouter *mux.Router) {
	// Create log instance for handling requests
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "VerrazzanoOperator").Str("name", "OperatorInit").Logger()

	apiRouter := registerPathHandlers(rootRouter)

	logger.Info().Msg("Starting API server...")

	if apiServerRealm == "" {
		logger.Warn().Msg("Bearer token verification is disabled as apiServerRealm is not specified")
	}

	logger.Fatal().Err(http.ListenAndServe(":3456", apiserver.CORSHandler(apiserver.AuthHandler(apiRouter))))
	apiServerFinished <- true
}

func registerPathHandlers(rootRouter *mux.Router) *mux.Router {
	apiRouter := rootRouter.PathPrefix(apiVersionPrefix).Subrouter()
	// All paths registered below are after path prefix specified in the apiVersionPrefix variable
	apiRouter.HandleFunc("/", homePage)

	// There is only 1
	apiRouter.Path("/instance").Methods("GET").HandlerFunc(instance.ReturnSingleInstance)

	// Routes are tested in the order they were added to the router. If two routes match, the first one wins
	apiRouter.Path("/applications").Methods("GET").HandlerFunc(applications.ReturnAllApplications)
	apiRouter.Path("/applications").Methods("POST").HandlerFunc(applications.CreateNewApplication)
	apiRouter.Path("/applications/{id}").Methods("GET").HandlerFunc(applications.ReturnSingleApplication)
	apiRouter.Path("/applications/{id}").Methods("DELETE").HandlerFunc(applications.DeleteApplication)
	apiRouter.Path("/applications/{id}").Methods("PUT").HandlerFunc(applications.UpdateApplication)

	apiRouter.Path("/clusters").Methods("GET").HandlerFunc(clusters.ReturnAllClusters)
	apiRouter.Path("/clusters/{id}").Methods("GET").HandlerFunc(clusters.ReturnSingleCluster)
	apiRouter.Path("/domains").Methods("GET").HandlerFunc(domains.ReturnAllDomains)
	apiRouter.Path("/domains/{id}").Methods("GET").HandlerFunc(domains.ReturnSingleDomain)

	apiRouter.Path("/grids").Methods("GET").HandlerFunc(grids.ReturnAllGrids)
	apiRouter.Path("/grids/{id}").Methods("GET").HandlerFunc(grids.ReturnSingleGrid)

	apiRouter.Path("/images").Methods("GET").HandlerFunc(images.ReturnAllImages)
	apiRouter.Path("/images/{id}").Methods("GET").HandlerFunc(images.ReturnSingleImage)

	apiRouter.Path("/jobs").Methods("GET").HandlerFunc(jobs.ReturnAllJobs)
	apiRouter.Path("/jobs/{id}").Methods("GET").HandlerFunc(jobs.ReturnSingleJob)

	apiRouter.Path("/microservices").Methods("GET").HandlerFunc(microservices.ReturnAllMicroservices)
	apiRouter.Path("/microservices/{id}").Methods("GET").HandlerFunc(microservices.ReturnSingleMicroservice)

	apiRouter.Path("/operators").Methods("GET").HandlerFunc(operators.ReturnAllOperators)
	apiRouter.Path("/operators/{id}").Methods("GET").HandlerFunc(operators.ReturnSingleOperator)

	apiRouter.Path("/secrets").Methods("GET").HandlerFunc(secrets.ReturnAllSecrets)
	apiRouter.Path("/secrets").Methods("POST").HandlerFunc(secrets.CreateSecret)
	apiRouter.Path("/secrets/{id}").Methods("GET").HandlerFunc(secrets.ReturnSingleSecret)
	apiRouter.Path("/secrets/{id}").Methods("DELETE").HandlerFunc(secrets.DeleteSecret)
	apiRouter.Path("/secrets/{id}").Methods("PATCH").HandlerFunc(secrets.UpdateSecret)

	return apiRouter
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

	logs.InitLogs()

	// Create log instance for main operator
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("kind", "VerrazzanoOperator").Str("name", "OperatorInit").Logger()

	manifest, err := util.LoadManifest()
	if err != nil {
		logger.Fatal().Msgf("Error loading manifest: %s", err.Error())
	}

	flag.Parse()


	logger.Debug().Msgf("Creating new controller watching namespace %s.", watchNamespace)
	if strings.EqualFold(helidonAppOperatorDeployment, "false") {
		logger.Debug().Msgf("helidonAppOperatorDeployment Disabled")
		manifest.HelidonAppOperatorImage = ""
	}
	controller, err := pkgverrazzanooperator.NewController(kubeconfig, manifest, masterURL, watchNamespace, verrazzanoURI, enableMonitoringStorage)
	if err != nil {
		logger.Fatal().Msgf("Error creating the controller: %s", err.Error())
	}

	apiServerExited := make(chan bool)

	// start the REST API Server (for web ui)
	startRestAPIServer(controller.ListerSet(), apiServerExited)

	if startController {
		// start the controller
		if err = controller.Run(2); err != nil {
			glog.Fatalf("Error running controller: %s", err.Error())
		}
	}

	glog.Info("Waiting for api server to exit")
	<-apiServerExited

}

func startRestAPIServer(listerSet pkgverrazzanooperator.Listers, apiServerFinished chan bool) {
	// give the handlers access to the informers
	applications.Init(listerSet)
	clusters.Init(listerSet)
	domains.Init(listerSet)
	grids.Init(listerSet)
	images.Init()
	jobs.Init()
	microservices.Init(listerSet)
	operators.Init(listerSet)
	secrets.Init(listerSet)
	apiserver.Init(listerSet)

	instance.SetVerrazzanoURI(verrazzanoURI)
	apiserver.SetRealm(apiServerRealm)
	// start REST handlers
	go handleRequests(apiServerFinished, mux.NewRouter().StrictSlash(true))
}

func init() {
	initFlags(flag.StringVar, flag.BoolVar)
}

// initFlags registers all command line flags. It accepts a registration function for string vars and
// one for boolean vars, to enable testing
func initFlags(
	stringVarFunc func(p *string, name string, value string, usage string),
	boolVarFunc func(p *bool, name string, value bool, usage string)) {

	stringVarFunc(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	stringVarFunc(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	stringVarFunc(&watchNamespace, "watchNamespace", "", "Optionally, a namespace to watch exclusively.  If not set, all namespaces will be watched.")
	stringVarFunc(&verrazzanoURI, "verrazzanoUri", "", "Verrazzano URI, for example my-verrazzano-1.verrazzano.example.com")
	stringVarFunc(&helidonAppOperatorDeployment, "helidonAppOperatorDeployment", "", "--helidonAppOperatorDeployment=false disables helidonAppOperatorDeployment")
	stringVarFunc(&enableMonitoringStorage, "enableMonitoringStorage", "", "Enable storage for monitoring.  The default is true. 'false' means monitoring storage is disabled.")
	stringVarFunc(&apiServerRealm, "apiServerRealm", "", "API Server Realm on Keycloak")
	boolVarFunc(&startController, "startController", true, "Whether to start the Kubernetes controller (true by default)")
}
