package worker

import (
	"context"
	"fmt"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type (
	QueueType workqueue.TypedRateLimitingInterface[string]
)

type Worker struct {
	kubeclient kubernetes.Interface
	workqueue  QueueType
}

func NewWorker(kubeclient kubernetes.Interface, queue QueueType) *Worker {
	return &Worker{
		kubeclient: kubeclient,
		workqueue:  queue,
	}
}

func (worker *Worker) Run() {
	for worker.process() {
	}
}

func (worker *Worker) process() bool {
	task, shutdown := worker.workqueue.Get()
	if shutdown {
		return false
	}

	defer worker.workqueue.Done(task)

	err := worker.refreshPod(task)
	if err != nil {
		worker.workqueue.AddRateLimited(task)
		klog.Errorf("cannot reconcile pods: %s", err)
		return true
	}

	worker.workqueue.Forget(task)

	return true
}

func (worker *Worker) refreshPod(key string) error {
	klog.Infof("processing eviction task: %s", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("splitting key: %s", err)
	}

	err = worker.kubeclient.CoreV1().Pods(namespace).
		Evict(context.Background(), &policyv1beta1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		})

	if err != nil {
		return fmt.Errorf("evicting pod: %s", err)
	}

	return nil
}
