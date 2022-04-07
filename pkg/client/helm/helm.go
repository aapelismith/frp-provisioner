package helm

import (
	"errors"
	"fmt"
	"github.com/spf13/pflag"
	"kunstack.com/pharos/pkg/safe"
	"os"
)

type Options struct {
	safe.NoCopy
	PodManifest        string `json:"podManifest,omitempty" yaml:"podManifest,omitempty"`
	ServiceAccountName string `json:"serviceAccountName,omitempty" yaml:"serviceAccountName,omitempty"`
}

func (o *Options) SetDefaults() {}

func (o *Options) Validate() error {
	if o.PodManifest == "" {
		return errors.New("helperPodFile is required filed")
	}

	_, err := os.Stat(o.PodManifest)
	if err != nil {
		return fmt.Errorf("unable stat file %s,got: %v", o.PodManifest, err)
	}

	if o.ServiceAccountName == "" {
		return errors.New("serviceAccountName is required filed")
	}

	return nil
}

func (o *Options) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	fs.StringVar(
		&o.PodManifest,
		"helm.pod-manifest",
		o.PodManifest,
		"Specify the path to the helper pod file",
	)

	fs.StringVar(
		&o.ServiceAccountName,
		"helm.service-account-name",
		o.ServiceAccountName,
		"Specifies the service account name to use when installing helm release",
	)
	return fs
}

func NewOptions() *Options {
	return &Options{
		PodManifest:        "",
		ServiceAccountName: "",
	}
}
