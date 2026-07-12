package cnpg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
)

var testGVR = schema.GroupVersionResource{
	Group:    "postgresql.cnpg.io",
	Version:  "v1",
	Resource: "clusters",
}

// liveCluster builds an unstructured Cluster carrying spec fields the scaler does
// not manage, so the test can assert they survive an annotation-only update.
func liveCluster(name, namespace string, annotations map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "postgresql.cnpg.io/v1",
			"kind":       "Cluster",
			"metadata": map[string]interface{}{
				"name":        name,
				"namespace":   namespace,
				"annotations": annotations,
			},
			"spec": map[string]interface{}{
				"instances": int64(3),
				"storage": map[string]interface{}{
					"size": "10Gi",
				},
			},
		},
	}
}

func TestClusterUpdater_PreservesUnmanagedSpecFields(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, AddToScheme(scheme))

	dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{testGVR: "ClusterList"},
		liveCluster("cluster-a", "test-ns", map[string]interface{}{"team": "platform"}),
	)
	client := dynClient.Resource(testGVR)

	getter := &clusterGetter{client: client}
	item, err := getter.Get(context.Background(), "test-ns", "cluster-a", metaV1.GetOptions{})
	require.NoError(t, err)

	// Simulate the strategy hibernating the cluster via annotations only.
	item.SetAnnotations(map[string]string{
		"team":                         "platform",
		base.CNPGHibernationAnnotation: base.CNPGHibernationOn,
	})

	updater := &clusterUpdater{client: client}
	_, err = updater.Update(context.Background(), "test-ns", item, metaV1.UpdateOptions{})
	require.NoError(t, err)

	live, err := client.Namespace("test-ns").Get(context.Background(), "cluster-a", metaV1.GetOptions{})
	require.NoError(t, err)

	annotations := live.GetAnnotations()
	assert.Equal(t, base.CNPGHibernationOn, annotations[base.CNPGHibernationAnnotation])
	assert.Equal(t, "platform", annotations["team"])

	// The unmanaged spec must be untouched.
	instances, found, err := unstructured.NestedInt64(live.Object, "spec", "instances")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, int64(3), instances)

	size, found, err := unstructured.NestedString(live.Object, "spec", "storage", "size")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "10Gi", size)
}

func TestClusterUpdater_RejectsForeignResourceItem(t *testing.T) {
	updater := &clusterUpdater{}
	_, err := updater.Update(context.Background(), "test-ns", foreignItem{}, metaV1.UpdateOptions{})

	require.Error(t, err)
	var typeErr *base.TypeAssertionError
	assert.ErrorAs(t, err, &typeErr)
}

// foreignItem is a ResourceItem that is not a *clusterItem, used to exercise the
// updater's type-assertion guard.
type foreignItem struct{}

func (foreignItem) GetName() string                   { return "foreign" }
func (foreignItem) GetNamespace() string              { return "test-ns" }
func (foreignItem) GetAnnotations() map[string]string { return nil }
func (foreignItem) SetAnnotations(map[string]string)  {}

func TestCnpg_Init_WiresClientAndAnnotationManager(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, AddToScheme(scheme))
	dynClient := dynamicfake.NewSimpleDynamicClient(scheme)

	c := &Cnpg{}
	c.init(dynClient)

	assert.NotNil(t, c.Client)
	assert.NotNil(t, c.AnnotationManager)
}

func TestCluster_DeepCopyObject(t *testing.T) {
	var nilCluster *Cluster
	assert.Nil(t, nilCluster.DeepCopyObject())

	original := &Cluster{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        "cluster-a",
			Annotations: map[string]string{base.CNPGHibernationAnnotation: base.CNPGHibernationOn},
		},
	}
	copied, ok := original.DeepCopyObject().(*Cluster)
	require.True(t, ok)
	copied.Annotations[base.CNPGHibernationAnnotation] = base.CNPGHibernationOff
	assert.Equal(t, base.CNPGHibernationOn, original.Annotations[base.CNPGHibernationAnnotation])
}

func TestClusterList_DeepCopyObject(t *testing.T) {
	var nilList *ClusterList
	assert.Nil(t, nilList.DeepCopyObject())

	original := &ClusterList{
		Items: []Cluster{
			{ObjectMeta: metaV1.ObjectMeta{Name: "cluster-a"}},
			{ObjectMeta: metaV1.ObjectMeta{Name: "cluster-b"}},
		},
	}
	copied, ok := original.DeepCopyObject().(*ClusterList)
	require.True(t, ok)
	require.Len(t, copied.Items, 2)
	copied.Items[0].Name = "mutated"
	assert.Equal(t, "cluster-a", original.Items[0].Name)
}
