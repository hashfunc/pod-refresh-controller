package main

import (
	"os"

	"k8s.io/klog/v2"

	pod_refresh_controller "github.com/hashfunc/pod-refresh-controller/pkg/controller"
	"github.com/hashfunc/pod-refresh-controller/pkg/kubeclient"
)

func main() {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		klog.Fatal("POD_NAME is not set")
	}

	podNamespace := os.Getenv("POD_NAMESPACE")
	if podNamespace == "" {
		klog.Fatal("POD_NAMESPACE is not set")
	}

	_, enableLocalConfig := os.LookupEnv("ENABLE_LOCAL_CONFIG")

	client, err := kubeclient.New(enableLocalConfig)
	if err != nil {
		klog.Fatalf("creating kubeclient: %s", err.Error())
	}

	controller := pod_refresh_controller.NewController(client, podName, podNamespace)

	stopCh := make(chan struct{})
	defer close(stopCh)

	if err := controller.Run(stopCh); err != nil {
		klog.Fatalf("running controller: %s", err.Error())
	}
}
