package controller

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type Controller struct {
	sharedInformerFactory informers.SharedInformerFactory
	podsSynced            cache.InformerSynced
}

func NewController(kubeclient kubernetes.Interface, namespace string) *Controller {
	sharedInformerFactory := informers.NewSharedInformerFactoryWithOptions(
		kubeclient,
		1*time.Minute,
		informers.WithNamespace(namespace),
	)

	podInformer := sharedInformerFactory.Core().V1().Pods()

	controller := &Controller{
		sharedInformerFactory: sharedInformerFactory,
		podsSynced:            podInformer.Informer().HasSynced,
	}

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, updated interface{}) {
			if pod, ok := updated.(*corev1.Pod); ok {
				klog.Infof("Pod updated: %s/%s", pod.Namespace, pod.Name)
			}
		},
	})

	return controller
}

func (controller *Controller) Run(stopCh <-chan struct{}) error {
	controller.sharedInformerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, controller.podsSynced); !ok {
		return fmt.Errorf("waiting for cache sync")
	}

	<-stopCh
	return nil
}
