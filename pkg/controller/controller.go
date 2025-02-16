package controller

import (
	"fmt"
	"runtime"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/hashfunc/pod-refresh-controller/pkg/worker"
)

type Controller struct {
	podName      string
	podNamespace string

	kubeclient            kubernetes.Interface
	sharedInformerFactory informers.SharedInformerFactory
	podLister             corelisters.PodLister
	podsSynced            cache.InformerSynced
	deploymentsSynced     cache.InformerSynced
	workqueue             worker.QueueType
	workers               []*worker.Worker
}

func NewController(kubeclient kubernetes.Interface, podName, podNamespace string) *Controller {
	sharedInformerFactory := informers.NewSharedInformerFactoryWithOptions(
		kubeclient,
		2*time.Minute,
		informers.WithNamespace(podNamespace),
	)

	podInformer := sharedInformerFactory.Core().V1().Pods()
	deploymentInformer := sharedInformerFactory.Apps().V1().Deployments()

	queue := workqueue.NewTypedRateLimitingQueue(
		workqueue.DefaultTypedControllerRateLimiter[*worker.EvictionTask](),
	)

	workers := make([]*worker.Worker, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		workers[i] = worker.NewWorker(queue)
	}

	controller := &Controller{
		podName:      podName,
		podNamespace: podNamespace,

		kubeclient:            kubeclient,
		sharedInformerFactory: sharedInformerFactory,
		podLister:             podInformer.Lister(),
		podsSynced:            podInformer.Informer().HasSynced,
		deploymentsSynced:     deploymentInformer.Informer().HasSynced,
		workqueue:             queue,
		workers:               workers,
	}

	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: controller.UpdateFuncDeployment,
	})

	return controller
}

func (controller *Controller) Run(stopCh <-chan struct{}) error {
	defer controller.workqueue.ShutDown()

	controller.sharedInformerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, controller.podsSynced, controller.deploymentsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Infof("starting workers(%d)", len(controller.workers))
	for i := range controller.workers {
		go controller.workers[i].Run()
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

	task := &worker.EvictionTask{
		Deployment: deployment,
		Pods:       pods,
	}

	controller.workqueue.Add(task)
}
