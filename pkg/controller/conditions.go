package controller

import (
	"fmt"
	"regexp"

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
