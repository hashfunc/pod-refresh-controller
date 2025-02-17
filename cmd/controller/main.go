package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"

	"github.com/hashfunc/pod-refresh-controller/pkg/config"
	pod_refresh_controller "github.com/hashfunc/pod-refresh-controller/pkg/controller"
	"github.com/hashfunc/pod-refresh-controller/pkg/kubeclient"
	"github.com/hashfunc/pod-refresh-controller/pkg/leaderelection"
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	configManager := config.NewManager(client, podNamespace)

	controller := pod_refresh_controller.NewController(
		client,
		podName,
		podNamespace,
		configManager.Config(),
		config.DefaultResyncPeriod,
	)

	if err := leaderelection.Run(ctx, client, podName, podNamespace, func(ctx context.Context) {
		configManager.Start(ctx.Done())

		if err := controller.Run(ctx.Done()); err != nil {
			klog.Fatalf("running controller: %s", err.Error())
		}
	}); err != nil {
		klog.Fatalf("error running leader election: %s", err.Error())
	}
}
