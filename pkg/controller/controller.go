package controller

import (
	"runtime"
	"time"

	appsv1 "k8s.io/api/apps/v1"
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

type Controller struct {
	podName      string
	podNamespace string
	config       *config.Config

	kubeclient            kubernetes.Interface
	sharedInformerFactory informers.SharedInformerFactory

	podLister         corelisters.PodLister
	podsSynced        cache.InformerSynced
	deploymentsSynced cache.InformerSynced

	workqueue       worker.QueueType
	worker          *worker.Worker
	numberOfWorkers int
}

func NewController(
	kubeclient kubernetes.Interface,
	podName, podNamespace string,
	config *config.Config,
	resyncPeriod time.Duration,
) *Controller {
	sharedInformerFactory := informers.NewSharedInformerFactoryWithOptions(
		kubeclient,
		resyncPeriod,
		informers.WithNamespace(podNamespace),
	)

	podInformer := sharedInformerFactory.Core().V1().Pods()
	deploymentInformer := sharedInformerFactory.Apps().V1().Deployments()

	queue := workqueue.NewTypedRateLimitingQueue(
		workqueue.DefaultTypedControllerRateLimiter[string](),
	)

	controller := &Controller{
		podName:      podName,
		podNamespace: podNamespace,
		config:       config,

		kubeclient:            kubeclient,
		sharedInformerFactory: sharedInformerFactory,

		podLister:         podInformer.Lister(),
		podsSynced:        podInformer.Informer().HasSynced,
		deploymentsSynced: deploymentInformer.Informer().HasSynced,

		workqueue:       queue,
		worker:          worker.NewWorker(kubeclient, queue),
		numberOfWorkers: runtime.NumCPU(),
	}

	_, _ = deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: controller.updateFuncDeployment,
	})

	return controller
}

func (controller *Controller) Run(stopCh <-chan struct{}) error {
	defer controller.workqueue.ShutDown()

	controller.sharedInformerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(
		stopCh,
		controller.podsSynced,
		controller.deploymentsSynced,
	); !ok {
		klog.Fatalf("waiting for cache sync")
	}

	klog.Infof("starting workers(%d)", controller.numberOfWorkers)

	for range controller.numberOfWorkers {
		go wait.Until(controller.worker.Run, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

func (controller *Controller) updateFuncDeployment(_, newObj interface{}) {
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
		if !controller.isPodExpired(pod) {
			continue
		}

		key, err := cache.MetaNamespaceKeyFunc(pod)
		if err != nil {
			klog.Errorf("cannot get key for %s: %s", pod.Name, err)

			continue
		}

		controller.workqueue.Add(key)
	}
}
