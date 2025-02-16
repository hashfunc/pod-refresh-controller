package worker

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type EvictionTask struct {
	Deployment *appsv1.Deployment
	Pods       []*corev1.Pod
}
