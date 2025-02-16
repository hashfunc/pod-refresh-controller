package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	pod_refresh_controller "github.com/hashfunc/pod-refresh-controller/pkg/controller"
)

func TestPodRefresh(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewSimpleClientset()

	podEviction := make(chan string, 1)

	fakeClient.PrependReactor(
		"create",
		"pods/eviction",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			eviction, ok := action.(k8stesting.CreateAction).GetObject().(*policyv1beta1.Eviction)
			if !ok {
				t.Fatalf("casting to eviction")
			}

			podEviction <- eviction.Name

			return true, nil, nil
		})

	deployment := getDeploymentForTesting()

	lo.Must(
		fakeClient.AppsV1().
			Deployments(deployment.Namespace).
			Create(context.Background(), deployment, metav1.CreateOptions{}))

	pod := getPodForTesting()

	lo.Must(
		fakeClient.CoreV1().
			Pods(pod.Namespace).
			Create(context.Background(), pod, metav1.CreateOptions{}))

	controller := pod_refresh_controller.NewController(
		fakeClient,
		"pod-refresh-controller",
		deployment.Namespace,
		2*time.Second,
	)

	stopCh := make(chan struct{})
	defer close(stopCh)

	go func() {
		lo.Must0(controller.Run(stopCh))
	}()

	select {
	case podName := <-podEviction:
		if podName != pod.Name {
			t.Errorf("expected pod %s to be evicted, got %s", pod.Name, podName)
		}
	case <-time.After(30 * time.Second):
		t.Error("timeout waiting for pod eviction")
	}
}

func getDeploymentForTesting() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testing",
			Namespace: "testing",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "testing",
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   appsv1.DeploymentProgressing,
					Status: corev1.ConditionTrue,
					Reason: "NewReplicaSetAvailable",
				},
			},
		},
	}
}

func getPodForTesting() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testing",
			Namespace: "testing",
			Labels: map[string]string{
				"app": "testing",
			},
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-25 * time.Hour)},
		},
	}
}
