package controller

import (
	"fmt"
	"regexp"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func (controller *Controller) isControllerDeployment(deploymentName string) bool {
	if deploymentName == "" {
		return false
	}

	pattern := fmt.Sprintf("^%s-[a-z0-9]+-[a-z0-9]+$", deploymentName)

	matched, err := regexp.MatchString(pattern, controller.podName)
	if err != nil {
		klog.Errorf("matching pod name pattern for %s: %s", deploymentName, err)
		return false
	}

	return matched
}

func (controller *Controller) isDeploymentReady(deployment *appsv1.Deployment) bool {
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status != corev1.ConditionTrue {
			return false
		}

		if condition.Type == appsv1.DeploymentProgressing {
			if condition.Status != corev1.ConditionTrue {
				return false
			}

			if condition.Reason != "NewReplicaSetAvailable" {
				return false
			}
		}
	}

	return true
}

func (controller *Controller) isPodExpired(pod *corev1.Pod) bool {
	return time.Since(pod.CreationTimestamp.Time) > controller.config.PodExpirationTime
}
