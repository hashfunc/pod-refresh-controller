package config

import (
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	klog "k8s.io/klog/v2"
)

type Manager struct {
	config *Config

	sharedInformerFactory informers.SharedInformerFactory
	configMapSynced       cache.InformerSynced
}

func getConfigMapName() string {
	configMapName := os.Getenv("CONFIG_MAP_NAME")

	if configMapName == "" {
		return DefaultConfigMapName
	}

	return configMapName
}

func NewManager(
	kubeclient kubernetes.Interface,
	podNamespace string,
) *Manager {
	sharedInformerFactory := informers.NewSharedInformerFactoryWithOptions(
		kubeclient,
		DefaultResyncPeriod,
		informers.WithNamespace(podNamespace),
		informers.WithTweakListOptions(
			func(options *metav1.ListOptions) {
				options.FieldSelector = "metadata.name=" + getConfigMapName()
			},
		),
	)

	configmapInformer := sharedInformerFactory.Core().V1().ConfigMaps()

	manager := &Manager{
		config: NewDefaultConfig(),

		sharedInformerFactory: sharedInformerFactory,
		configMapSynced:       configmapInformer.Informer().HasSynced,
	}

	_, _ = configmapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: manager.updateConfig,
		UpdateFunc: func(_, newObj interface{}) {
			manager.updateConfig(newObj)
		},
	})

	return manager
}

func (manager *Manager) Start(stopCh <-chan struct{}) {
	manager.sharedInformerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, manager.configMapSynced); !ok {
		klog.Fatalf("waiting for cache sync(configmap)")
	}

	<-stopCh
}

func (manager *Manager) Config() *Config {
	return manager.config
}

func (manager *Manager) updateConfig(obj interface{}) {
	configmap, ok := obj.(*corev1.ConfigMap)
	if !ok {
		klog.Errorf("casting to configmap: %T", obj)
		return
	}

	podExpirationTime, err := time.ParseDuration(configmap.Data["podExpirationTime"])
	if err != nil {
		klog.Errorf("parsing pod expiration time: %s", err)
		return
	}

	if manager.config.PodExpirationTime != podExpirationTime {
		klog.Infof("pod expiration time updated to %s", podExpirationTime)
		manager.config.PodExpirationTime = podExpirationTime
	}
}
