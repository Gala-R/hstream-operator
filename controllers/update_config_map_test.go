package controllers

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("UpdateConfigMap", func() {
	var hdb *appsv1alpha1.HStreamDB
	var requeue *requeue
	updateConfigMap := updateConfigMap{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())

		requeue = updateConfigMap.reconcile(ctx, clusterReconciler, hdb)
	})

	AfterEach(func() {
		k8sClient.Delete(ctx, hdb)
	})

	Context("with a reconciled cluster", func() {
		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		var logDevice, nShard *corev1.ConfigMap
		It("should successfully get config map", func() {
			var err error
			logDevice, nShard, err = getConfigMaps(hdb)
			Expect(err).To(BeNil())
		})

		When("config maps have been deploy", func() {
			Context("reconcile though nothing change", func() {
				BeforeEach(func() {
					requeue = updateConfigMap.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get same uid", func() {
					newLogDevice, newNShards, err := getConfigMaps(hdb)
					Expect(err).To(BeNil())
					Expect(logDevice.UID).To(Equal(newLogDevice.UID))
					Expect(nShard.UID).To(Equal(newNShards.UID))
				})
			})

			Context("update config name", func() {
				BeforeEach(func() {
					nshard := int32(2)
					hdb.Spec.Config.NShards = &nshard
					hdb.Spec.Config.LogDeviceConfig = runtime.RawExtension{
						Raw: []byte(`
					{
						"server_settings": {
						  "enable-nodes-configuration-manager": "false",
						  "use-nodes-configuration-manager-nodes-configuration": "false",
						  "enable-node-self-registration": "false",
						  "enable-cluster-maintenance-state-machine": "false"
						}
					}
					`)}
					requeue = updateConfigMap.reconcile(ctx, clusterReconciler, hdb)
					Expect(requeue).To(BeNil())
				})

				var newLogDevice, newNShard *corev1.ConfigMap
				It("should get new config map", func() {
					var err error
					newLogDevice, newNShard, err = getConfigMaps(hdb)
					Expect(err).To(BeNil())
				})

				It("should get new server_setting", func() {
					cm, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
					Expect(newLogDevice.Data).To(HaveKey(cm.MapKey))
					file := newLogDevice.Data[cm.MapKey]
					m := make(map[string]any)
					err := json.UnmarshalFromString(file, &m)
					Expect(err).To(BeNil())
					Expect(m).To(HaveKeyWithValue("server_settings", map[string]any{
						"enable-nodes-configuration-manager":                  "false",
						"use-nodes-configuration-manager-nodes-configuration": "false",
						"enable-node-self-registration":                       "false",
						"enable-cluster-maintenance-state-machine":            "false",
					}))
				})

				It("should get new nShard", func() {
					cm, _ := internal.ConfigMaps.Get(internal.NShardsConfig)
					Expect(newNShard.Data).To(HaveKey(cm.MapKey))
					Expect(newNShard.Data).To(HaveKeyWithValue("NSHARDS", "2"))
				})
			})
		})
	})
})

func getConfigMaps(hdb *appsv1alpha1.HStreamDB) (logDevice, nShards *corev1.ConfigMap, err error) {
	config, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
	keyObj := types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      internal.GetResNameOnPanic(hdb, config.MapNameSuffix),
	}
	logDevice = &corev1.ConfigMap{}
	if err = k8sClient.Get(context.TODO(), keyObj, logDevice); err != nil {
		err = fmt.Errorf("get log device config failed: %w", err)
		return
	}

	config, _ = internal.ConfigMaps.Get(internal.NShardsConfig)
	keyObj.Name = internal.GetResNameOnPanic(hdb, config.MapNameSuffix)
	nShards = &corev1.ConfigMap{}
	if err = k8sClient.Get(context.TODO(), keyObj, nShards); err != nil {
		err = fmt.Errorf("get nshard config failed: %w", err)
		return
	}
	return
}
