// Package service implements business logic to create Kubernetes resources
// against the Kubernetes API.
package service

import (
	"context"
	"sync"

	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v3/pkg/k8srestconfig"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/rbac-operator/flag"
	"github.com/giantswarm/rbac-operator/pkg/project"
	"github.com/giantswarm/rbac-operator/service/collector"
	"github.com/giantswarm/rbac-operator/service/controller/namespacelabeler"
	"github.com/giantswarm/rbac-operator/service/controller/rbac"

	"github.com/giantswarm/rbac-operator/service/controller/rbac/resource/namespaceauth"
)

// Config represents the configuration used to create a new service.
type Config struct {
	Logger micrologger.Logger

	Flag  *flag.Flag
	Viper *viper.Viper
}

type Service struct {
	Version *version.Service

	bootOnce                   sync.Once
	namespaceLabelerController *namespacelabeler.NamespaceLabeler
	rbacController             *rbac.RBAC
	operatorCollector          *collector.Set
}

// New creates a new configured service object.
func New(config Config) (*Service, error) {
	var serviceAddress string
	// Settings.
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Viper must not be empty")
	}
	if config.Flag.Service.Kubernetes.KubeConfig == "" {
		serviceAddress = config.Viper.GetString(config.Flag.Service.Kubernetes.Address)
	} else {
		serviceAddress = ""
	}

	// Dependencies.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "logger must not be empty")
	}

	var err error

	var restConfig *rest.Config
	{
		c := k8srestconfig.Config{
			Logger: config.Logger,

			Address:    serviceAddress,
			InCluster:  config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster),
			KubeConfig: config.Viper.GetString(config.Flag.Service.Kubernetes.KubeConfig),
			TLS: k8srestconfig.ConfigTLS{
				CAFile:  config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile),
				CrtFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile),
				KeyFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile),
			},
		}

		restConfig, err = k8srestconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var k8sClient k8sclient.Interface
	{
		c := k8sclient.ClientsConfig{
			Logger:     config.Logger,
			RestConfig: restConfig,
		}

		k8sClient, err = k8sclient.NewClients(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var rbacController *rbac.RBAC
	{

		c := rbac.RBACConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,

			NamespaceAuth: namespaceauth.NamespaceAuth{
				ViewAllTargetGroup:     config.Viper.GetString(config.Flag.Service.NamespaceAuth.ViewAllTargetGroup),
				TenantAdminTargetGroup: config.Viper.GetString(config.Flag.Service.NamespaceAuth.TenantAdminTargetGroup),
			},
		}

		rbacController, err = rbac.NewRBAC(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceLabelerController *namespacelabeler.NamespaceLabeler
	{

		c := namespacelabeler.NamespaceLabelerConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,
		}

		namespaceLabelerController, err = namespacelabeler.NewNamespaceLabeler(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorCollector *collector.Set
	{
		c := collector.SetConfig{
			K8sClient: k8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		operatorCollector, err = collector.NewSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var versionService *version.Service
	{
		c := version.Config{
			Description: project.Description(),
			GitCommit:   project.GitSHA(),
			Name:        project.Name(),
			Source:      project.Source(),
			Version:     project.Version(),
		}

		versionService, err = version.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	s := &Service{
		Version: versionService,

		bootOnce:                   sync.Once{},
		namespaceLabelerController: namespaceLabelerController,
		rbacController:             rbacController,
		operatorCollector:          operatorCollector,
	}

	return s, nil
}

func (s *Service) Boot(ctx context.Context) {
	s.bootOnce.Do(func() {
		go s.operatorCollector.Boot(ctx)
		go s.namespaceLabelerController.Boot(ctx)

		go s.rbacController.Boot(ctx)
	})
}
