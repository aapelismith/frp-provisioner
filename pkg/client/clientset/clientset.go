package clientset

import (
	"errors"
	"fmt"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kunstack.com/pharos/pkg/safe"
	"os"
	"path/filepath"
)

type Options struct {
	safe.NoCopy
	KubeConfig string `yaml:"kube_config,omitempty"`
}

func (o *Options) Validate() error {
	_, err := rest.InClusterConfig()
	if err != nil {
		if o.KubeConfig == "" {
			return errors.New("kube_config field is required when not running in a k8s cluster")
		}

		_, err := os.Stat(o.KubeConfig)
		if err != nil {
			return fmt.Errorf("unable stat file %s,got: %v", o.KubeConfig, err)
		}
	}
	return nil
}

func (o *Options) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	fs.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Absolute path to the kubeconfig file(optional)")
	return fs
}

// NewOptions for creating a new clientSet
func NewOptions() *Options {
	var kubeConfig string
	if home, _ := os.UserHomeDir(); home != "" {
		kubeConfig = filepath.Join(home, ".kube", "config")
	}
	return &Options{KubeConfig: kubeConfig}
}

func NewClient(opt *Options) (*kubernetes.Clientset, error) {
	// 使用 ServiceAccount 创建集群配置（InCluster模式）
	config, err := NewConfig(opt)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func NewConfig(opt *Options) (*rest.Config, error) {
	// 使用 ServiceAccount 创建集群配置（InCluster模式）
	config, err := rest.InClusterConfig()
	if err == nil {
		// 使用 KubeConfig 文件创建集群配置
		if config, err = clientcmd.BuildConfigFromFlags("", opt.KubeConfig); err != nil {
			return nil, err
		}
	}
	return config, nil
}
