package loadbalancer

import (
	"errors"
	"fmt"
	"github.com/spf13/pflag"
	"kunstack.com/pharos/pkg/safe"
	"kunstack.com/pharos/pkg/types"
	"regexp"
)

var (
	_            types.Configurator = (*Options)(nil)
	domainRegexp                    = regexp.MustCompile(`^(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,255})+$`)
)

// Options for creating a new load balancing controller
type Options struct {
	safe.NoCopy
	EdgeImage        string   `yaml:"edge_image,omitempty"`
	ImagePullSecrets []string `yaml:"image_pull_secrets,omitempty"`
	DomainSuffix     string   `yaml:"domain_suffix,omitempty"`
}

func (o *Options) Validate() error {
	if o.EdgeImage == "" {
		return errors.New("edge_image is required field")
	}

	if o.DomainSuffix == "" {
		return errors.New("domain_suffix is required field")
	}

	if !domainRegexp.MatchString(o.DomainSuffix) {
		return fmt.Errorf("incorrect domain_suffix format")
	}

	return nil
}

func (o *Options) SetDefaults() {
	o.EdgeImage = "rancher/klipper-lb:v0.1.2"
}

func (o *Options) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	fs.StringVar(&o.DomainSuffix, "loadbalancer.domain-suffix", o.DomainSuffix, "Domain name suffix of the load balancer")
	fs.StringVar(&o.EdgeImage, "loadbalancer.edge-image", o.EdgeImage, "The image address of the container running on the edge node")
	fs.StringArrayVar(&o.ImagePullSecrets, "loadbalancer.image-pull-secrets", o.ImagePullSecrets, "Specify the k8s secret used to pull the image")
	return fs
}

// NewOptions for creating a new load balancing controller
func NewOptions() *Options {
	return &Options{}
}
