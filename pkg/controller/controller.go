package controller

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type Controller struct {
	kubeclient            kubernetes.Interface
	sharedInformerFactory informers.SharedInformerFactory
	podLister             corelisters.PodLister
	podsSynced            cache.InformerSynced
	deploymentsSynced     cache.InformerSynced
}

func NewController(kubeclient kubernetes.Interface, namespace string) *Controller {
	sharedInformerFactory := informers.NewSharedInformerFactoryWithOptions(
		kubeclient,
		1*time.Minute,
		informers.WithNamespace(namespace),
	)

	podInformer := sharedInformerFactory.Core().V1().Pods()
	deploymentInformer := sharedInformerFactory.Apps().V1().Deployments()

	controller := &Controller{
		kubeclient:            kubeclient,
		sharedInformerFactory: sharedInformerFactory,
		podLister:             podInformer.Lister(),
		podsSynced:            podInformer.Informer().HasSynced,
		deploymentsSynced:     deploymentInformer.Informer().HasSynced,
	}

	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: controller.UpdateFuncDeployment,
	})

	return controller
}

func (c *Controller) UpdateFuncDeployment(oldObj, newObj interface{}) {
	deployment, ok := newObj.(*appsv1.Deployment)
	if !ok {
		klog.Errorf("casting to deployment: %T", newObj)
		return
	}

	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		klog.Errorf("parsing selector: %s", err)
		return
	}

	pods, err := c.podLister.Pods(deployment.Namespace).List(selector)
	if err != nil {
		klog.Errorf("listing pods for %s: %s", deployment.Name, err)
		return
	}

	for _, pod := range pods {
		klog.Infof("%s created at %s", pod.Name, pod.CreationTimestamp)
	}
}

func (controller *Controller) Run(stopCh <-chan struct{}) error {
	controller.sharedInformerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, controller.podsSynced, controller.deploymentsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	<-stopCh
	return nil
}
