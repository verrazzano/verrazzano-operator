// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/verrazzano/verrazzano-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/labels"
)

type Cluster struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	ServerAddress string `json:"serverAddress"`
	Description   string `json:"description"`
	Status        string `json:"status"`
}

var (
	Clusters  []Cluster
	listerSet controller.Listers
)

func Init(listers controller.Listers) {
	listerSet = listers
	refreshClusters()
}

func GetClusters() ([]Cluster, error) {
	err := refreshClusters()
	if err != nil {
		return nil, err
	}
	return Clusters, nil
}

func refreshClusters() error {

	clusterSelector := labels.SelectorFromSet(map[string]string{})
	managedClusters, err := (*listerSet.ManagedClusterLister).VerrazzanoManagedClusters("default").List(clusterSelector)
	if err != nil {
		glog.Errorf("Error failed getting managed clusters: %s", err.Error())
		return err
	}

	Clusters = []Cluster{}
	i := 0
	for _, cluster := range managedClusters {
		Clusters = append(Clusters, Cluster{
			Id:            string(cluster.UID),
			Name:          cluster.Name,
			Type:          cluster.Spec.Type,
			ServerAddress: cluster.Spec.ServerAddress,
			Description:   cluster.Spec.Description,
			Status:        "OK",
		})
		i++
	}
	return nil
}

func ReturnAllClusters(w http.ResponseWriter, r *http.Request) {
	glog.V(4).Info("GET /clusters")

	err := refreshClusters()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting managed clusters: %s ", err.Error()),
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Total-Count", strconv.FormatInt(int64(len(Clusters)), 10))
	json.NewEncoder(w).Encode(Clusters)
}

func ReturnSingleCluster(w http.ResponseWriter, r *http.Request) {
	err := refreshClusters()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting managed clusters: %s", err.Error()),
			http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	key := vars["id"]

	glog.V(4).Info("GET /clusters/" + key)

	// Loop over all of our Clusters
	// if the article.Id equals the key we pass in
	// return the article encoded as JSON
	for _, clusters := range Clusters {
		if clusters.Id == key {
			json.NewEncoder(w).Encode(clusters)
		}
	}
}
