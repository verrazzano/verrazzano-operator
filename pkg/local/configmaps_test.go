// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package local

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	v1beta1v8o "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/constants"
)

func TestNewConfigMap(t *testing.T) {
	type args struct {
		binding *v1beta1v8o.VerrazzanoBinding
	}
	tests := []struct {
		name         string
		args         args
		expectedSize int
		wantErr      bool
	}{
		{
			name: "system",
			args: args{
				&v1beta1v8o.VerrazzanoBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "system",
					}},
			},
			expectedSize: 2,
		}, {
			name: "bookstore",
			args: args{
				&v1beta1v8o.VerrazzanoBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bookstore",
					}},
			},
			expectedSize: 19,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := newConfigMap(tt.args.binding)
			if (err != nil) != tt.wantErr {
				t.Errorf("newConfigMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.expectedSize, len(cm.Data), "Expected size of Data")
		})
	}
}

type Dashboard struct {
	Title  string  `json:"title"`
	Panels []Panel `json:"panels"`
}

type Panel struct {
	Title  string  `json:"title"`
	Type   string  `json:"type"`
	Id     int     `json:"id"`
	Panels []Panel `json:"panels"`
}

func TestSystemDashboard(t *testing.T) {
	sysDash := constants.SystemDashboards[1]
	//"Need to run "make go-install" to build assets
	//content, err := assets.Asset(sysDash)
	filename, _ := filepath.Abs(fmt.Sprintf("../../%s", sysDash))
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Error reading %v %v", sysDash, err)
	}
	var dashboard Dashboard
	err = json.Unmarshal(content, &dashboard)
	if err != nil {
		t.Fatalf("Error reading %v %v", sysDash, err)
	}
	ids := map[int]*[]string{0: {fmt.Sprintf("'%v'(%v)", dashboard.Title, 0)}}
	for _, p := range dashboard.Panels {
		ids = checkPanelId(t, 0, ids, &p)
	}
	panCPU := findPanel(201, dashboard.Panels)
	assert.Equal(t, "CPU", panCPU.Title, "panelCPU.Title")
	assert.Equal(t, 2, len(panCPU.Panels), "panelCPU.Panels size")
	assert.Equal(t, "table", panCPU.Panels[0].Type, "panelCPU type")
	assert.Equal(t, "graph", panCPU.Panels[1].Type, "panelCPU type")
	panCPU = findPanel(202, dashboard.Panels)
	assert.Equal(t, "CPU of $OriginalInstance", panCPU.Title, "panelCPU.Title")
	assert.Equal(t, 2, len(panCPU.Panels), "panelCPU.Panels size")
	assert.Equal(t, "Host CPU Usage", panCPU.Panels[0].Title, "panelCPU.Title")
	assert.Equal(t, "vCPU Usage", panCPU.Panels[1].Title, "panelCPU.Title")
}

func checkPanelId(t *testing.T, parentId int, ids map[int]*[]string, p *Panel) map[int]*[]string {
	parentPath := ids[parentId]
	path := make([]string, len(*parentPath))
	copy(path, *parentPath)
	path = append(path, fmt.Sprintf("'%v'(%v)", p.Title, p.Id))
	existingPanel := ids[p.Id]
	if existingPanel != nil {
		t.Fatalf("Duplicate Panel id %v %v", existingPanel, path)
	} else {
		ids[p.Id] = &path
	}
	if p.Panels != nil {
		for _, child := range p.Panels {
			ids = checkPanelId(t, p.Id, ids, &child)
		}
	}
	return ids
}

func findPanel(id int, panels []Panel) *Panel {
	for _, p := range panels {
		if p.Id == id {
			return &p
		}
		if p.Panels != nil {
			found := findPanel(id, p.Panels)
			if found != nil {
				return found
			}
		}
	}
	return nil
}
