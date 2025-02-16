package kubeclient

import (
	"fmt"
	"os"
	"path"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func New(enableLocalConfig bool) (*kubernetes.Clientset, error) {
	var config *rest.Config

	if enableLocalConfig {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting user home directory: %w", err)
		}

		kubeconfig := path.Join(homeDir, ".kube", "config")

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("building from local config: %w", err)
		}
	} else {
		var err error

		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("building in-cluster config: %w", err)
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}

	return client, nil
}
