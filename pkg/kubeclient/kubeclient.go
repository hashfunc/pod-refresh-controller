package kubeclient

import (
	"os"
	"path"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func New(enableLocalConfig bool) mo.Result[*kubernetes.Clientset] {
	return mo.Do(func() *kubernetes.Clientset {
		config := lo.TernaryF(enableLocalConfig, fromLocalConfig, inClusterConfig).MustGet()

		return lo.Must(kubernetes.NewForConfig(config))
	})
}

func fromLocalConfig() mo.Result[*rest.Config] {
	return mo.Do(func() *rest.Config {
		homeDir := lo.Must(os.UserHomeDir())

		kubeconfig := path.Join(homeDir, ".kube", "config")

		return lo.Must(clientcmd.BuildConfigFromFlags("", kubeconfig))
	})
}

func inClusterConfig() mo.Result[*rest.Config] {
	return mo.TupleToResult(rest.InClusterConfig())
}
