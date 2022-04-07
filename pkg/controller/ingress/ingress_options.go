package ingress

import (
	"errors"
	"fmt"
	"github.com/spf13/pflag"
	"kunstack.com/pharos/pkg/safe"
	"regexp"
)

var (
	domainRegexp = regexp.MustCompile(`^(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,255})+$`)
)

// Options for creating a new load balancing controller
type Options struct {
	safe.NoCopy
	DomainSuffix      string `yaml:"domainSuffix,omitempty"`
	DaemonSetManifest string `yaml:"daemonSetManifest,omitempty"`
}

func (o *Options) Validate() error {
	if o.DomainSuffix == "" {
		return errors.New("domainSuffix is required field")
	}

	if !domainRegexp.MatchString(o.DomainSuffix) {
		return fmt.Errorf("incorrect domainSuffix format")
	}

	return nil
}

func (o *Options) SetDefaults() {}

func (o *Options) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	fs.StringVar(&o.DomainSuffix, "ingress.domain-suffix", o.DomainSuffix, "Specify the k8s secret used to pull the image")
	fs.StringVar(&o.DaemonSetManifest, "ingress.daemonset-manifest", o.DaemonSetManifest, "Specifies the path or content of the daemonSet's manifest file")
	return fs
}

// NewOptions for creating a new load balancing controller
func NewOptions() *Options {
	return &Options{}
}
