package leaderelection

import (
	"context"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

const (
	DefaultLeaseDuration = 15 * time.Second
	DefaultRenewDeadline = 10 * time.Second
	DefaultRetryPeriod   = 2 * time.Second
)

func Run(ctx context.Context, client kubernetes.Interface, podName, podNamespace string, run func(context.Context)) error {
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "pod-refresh-controller",
			Namespace: podNamespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: podName,
		},
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   DefaultLeaseDuration,
		RenewDeadline:   DefaultRenewDeadline,
		RetryPeriod:     DefaultRetryPeriod,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Infof("became leader, starting controller")
				run(ctx)
			},
			OnStoppedLeading: func() {
				klog.Info("stopped leading")
				os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				if identity == podName {
					klog.Info("still the leader!")
					return
				}
				klog.Infof("new leader is %s", identity)
			},
		},
	})

	return nil
}
