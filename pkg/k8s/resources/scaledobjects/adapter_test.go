package scaledobjects

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncSpecReplicas_patchesOnlyReplicaFields(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"minReplicaCount": int64(2),
				"maxReplicaCount": int64(10),
				"scaleTargetRef": map[string]interface{}{
					"name": "my-deployment",
				},
				"triggers": []interface{}{
					map[string]interface{}{"type": "prometheus"},
				},
				"pollingInterval": int64(30),
			},
		},
	}

	spec := ScaledObjectSpec{
		MinReplicaCount: ptr.To(int32(0)),
		MaxReplicaCount: ptr.To(int32(5)),
	}

	require.NoError(t, syncSpecReplicas(spec, obj))

	specMap, ok := obj.Object["spec"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, int64(0), specMap["minReplicaCount"], "minReplicaCount must be updated")
	assert.Equal(t, int64(5), specMap["maxReplicaCount"], "maxReplicaCount must be updated")
	assert.Equal(t, map[string]interface{}{"name": "my-deployment"}, specMap["scaleTargetRef"], "scaleTargetRef must be preserved")
	assert.Len(t, specMap["triggers"], 1, "triggers must be preserved")
	assert.Equal(t, int64(30), specMap["pollingInterval"], "pollingInterval must be preserved")
}

func TestSyncSpecReplicas_removesFieldWhenNil(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"minReplicaCount": int64(2),
				"maxReplicaCount": int64(10),
			},
		},
	}

	spec := ScaledObjectSpec{
		MinReplicaCount: nil,
		MaxReplicaCount: nil,
	}

	require.NoError(t, syncSpecReplicas(spec, obj))

	specMap, ok := obj.Object["spec"].(map[string]interface{})
	require.True(t, ok)
	assert.NotContains(t, specMap, "minReplicaCount")
	assert.NotContains(t, specMap, "maxReplicaCount")
}
