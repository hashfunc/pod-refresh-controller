package controller

import (
	"fmt"
	"runtime"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/hashfunc/pod-refresh-controller/pkg/config"
	"github.com/hashfunc/pod-refresh-controller/pkg/worker"
)

const (
	DefaultResyncPeriod = 2 * time.Minute
)

type Controller struct {
	podName      string
	podNamespace string
	config       *config.Config

	kubeclient                        kubernetes.Interface
	sharedInformerFactory             informers.SharedInformerFactory
	sharedInformerFactoryForConfigMap informers.SharedInformerFactory

	podLister         corelisters.PodLister
	podsSynced        cache.InformerSynced
	deploymentsSynced cache.InformerSynced
	configmapsSynced  cache.InformerSynced

	workqueue       worker.QueueType
	worker          *worker.Worker
	numberOfWorkers int
}

func NewController(kubeclient kubernetes.Interface, podName, podNamespace string) *Controller {
	sharedInformerFactory := informers.NewSharedInformerFactoryWithOptions(
		kubeclient,
		DefaultResyncPeriod,
		informers.WithNamespace(podNamespace),
	)

	sharedInformerFactoryForConfigMap := informers.NewSharedInformerFactoryWithOptions(
		kubeclient,
		DefaultResyncPeriod,
		informers.WithNamespace(podNamespace),
		informers.WithTweakListOptions(
			func(options *metav1.ListOptions) {
				options.FieldSelector = fmt.Sprintf("metadata.name=%s", config.DefaultConfigMapName)
			},
		),
	)

	podInformer := sharedInformerFactory.Core().V1().Pods()
	deploymentInformer := sharedInformerFactory.Apps().V1().Deployments()
	configmapInformer := sharedInformerFactoryForConfigMap.Core().V1().ConfigMaps()

	queue := workqueue.NewTypedRateLimitingQueue(
		workqueue.DefaultTypedControllerRateLimiter[string](),
	)

	controller := &Controller{
		podName:      podName,
		podNamespace: podNamespace,
		config:       config.NewDefaultConfig(),

		kubeclient:                        kubeclient,
		sharedInformerFactory:             sharedInformerFactory,
		sharedInformerFactoryForConfigMap: sharedInformerFactoryForConfigMap,

		podLister:         podInformer.Lister(),
		podsSynced:        podInformer.Informer().HasSynced,
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		configmapsSynced:  configmapInformer.Informer().HasSynced,

		workqueue:       queue,
		worker:          worker.NewWorker(queue),
		numberOfWorkers: runtime.NumCPU(),
	}

	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: controller.UpdateFuncDeployment,
	})

	configmapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.UpdateConfig,
		UpdateFunc: func(_, newObj interface{}) {
			controller.UpdateConfig(newObj)
		},
	})

	return controller
}

func (controller *Controller) Run(stopCh <-chan struct{}) error {
	defer controller.workqueue.ShutDown()

	controller.sharedInformerFactory.Start(stopCh)
	controller.sharedInformerFactoryForConfigMap.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, controller.podsSynced, controller.deploymentsSynced, controller.configmapsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Infof("starting workers(%d)", controller.numberOfWorkers)
	for i := 0; i < controller.numberOfWorkers; i++ {
		go wait.Until(controller.worker.Run, time.Second, stopCh)
	}

	<-stopCh
	return nil
}

func (controller *Controller) UpdateFuncDeployment(oldObj, newObj interface{}) {
	deployment, ok := newObj.(*appsv1.Deployment)
	if !ok {
		klog.Errorf("casting to deployment: %T", newObj)
		return
	}

	if controller.isControllerDeployment(deployment.Name) {
		return
	}

	if !controller.isDeploymentReady(deployment) {
		klog.Infof("cannot reconcile pods: %s is not ready", deployment.Name)
		return
	}

	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		klog.Errorf("parsing selector: %s", err)
		return
	}

	pods, err := controller.podLister.Pods(deployment.Namespace).List(selector)
	if err != nil {
		klog.Errorf("listing pods for %s: %s", deployment.Name, err)
		return
	}

	for _, pod := range pods {
		key, err := cache.MetaNamespaceKeyFunc(pod)
		if err != nil {
			klog.Errorf("cannot get key for %s: %s", pod.Name, err)
			continue
		}

		controller.workqueue.Add(key)
	}
}

func (controller *Controller) UpdateConfig(obj interface{}) {
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

	klog.Infof("pod expiration time: %s", podExpirationTime)

	if controller.config.PodExpirationTime != podExpirationTime {
		klog.Infof("pod expiration time updated to %s", podExpirationTime)
		controller.config.PodExpirationTime = podExpirationTime
	}
}
