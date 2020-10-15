package fluentd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateGenericFluentdConfigMap(t *testing.T) {
	assertion := assert.New(t)

	labels := map[string]string{
		"myLabel": "hello",
	}
	cm := CreateFluentdConfigMap(GenericComponentFluentdConfiguration, "test-generic", "test-namespace", labels)
	assertion.Equal("test-generic-fluentd", cm.Name, "config map name not equal to expected value")
	assertion.Equal("test-namespace", cm.Namespace, "config map namespace not equal to expected value")
	assertion.Equal(labels, cm.Labels, "config map labels not equal to expected value")
	assertion.Equal(1, len(cm.Data), "config map data size not equal to expected value")
	assertion.NotNil(cm.Data[fluentdConf], "Expected fluentd.conf")
	assertion.Equal(cm.Data[fluentdConf], GenericComponentFluentdConfiguration, "config map data not equal to expected value")
}

func TestCreateHelidonFluentdConfigMap(t *testing.T) {
	assertion := assert.New(t)

	labels := map[string]string{
		"myLabel": "hello",
	}
	cm := CreateFluentdConfigMap(HelidonFluentdConfiguration, "test-helidon", "test-namespace", labels)
	assertion.Equal("test-helidon-fluentd", cm.Name, "config map name not equal to expected value")
	assertion.Equal("test-namespace", cm.Namespace, "config map namespace not equal to expected value")
	assertion.Equal(labels, cm.Labels, "config map labels not equal to expected value")
	assertion.Equal(1, len(cm.Data), "config map data size not equal to expected value")
	assertion.NotNil(cm.Data[fluentdConf], "Expected fluentd.conf")
	assertion.Equal(cm.Data[fluentdConf], HelidonFluentdConfiguration, "config map data not equal to expected value")
}
