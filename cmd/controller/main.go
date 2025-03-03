package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"

	"github.com/hashfunc/pod-refresh-controller/pkg/common"
	"github.com/hashfunc/pod-refresh-controller/pkg/config"
	pod_refresh_controller "github.com/hashfunc/pod-refresh-controller/pkg/controller"
	"github.com/hashfunc/pod-refresh-controller/pkg/kubeclient"
	"github.com/hashfunc/pod-refresh-controller/pkg/leaderelection"
)

func main() {
	podName := common.GetEnv("POD_NAME").MustGet()
	podNamespace := common.GetEnv("POD_NAMESPACE").MustGet()

	enableLocalConfig := common.GetEnv("ENABLE_LOCAL_CONFIG").IsPresent()

	client := kubeclient.New(enableLocalConfig).MustGet()

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
