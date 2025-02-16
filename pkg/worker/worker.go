package worker

import (
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type (
	QueueType workqueue.TypedRateLimitingInterface[*EvictionTask]
)

type Worker struct {
	workqueue QueueType
}

func NewWorker(queue QueueType) *Worker {
	return &Worker{
		workqueue: queue,
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

	err := worker.EvictionTaskHandler(task)
	if err != nil {
		worker.workqueue.AddRateLimited(task)
		klog.Errorf("cannot reconcile pods: %s", err)
		return true
	}

	worker.workqueue.Forget(task)

	return true
}

func (worker *Worker) EvictionTaskHandler(task *EvictionTask) error {
	klog.Infof("processing eviction task: %s", task.Deployment.Name)
	for _, pod := range task.Pods {
		klog.Infof("target pod: %s", pod.Name)
	}

	return nil
}
