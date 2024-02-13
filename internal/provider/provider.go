// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kaplan-michael/terraform-provider-olm/internal/olm/installer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// New is the factory function to return the provider.Provider implementation.
// It's best practice to instantiate the provider once and configure it using
// the Configure method.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OLMProvider{
			version: version,
		}
	}
}

// OLMProvider defines the provider implementation.
type OLMProvider struct {
	version string
	config  *OLMProviderModel
	client  *installer.Client
}

// Ensure OLMProvider implements the provider.Provider interface.
var _ provider.Provider = &OLMProvider{}

// OLMProviderModel describes the provider configuration.
type OLMProviderModel struct {
	Kubeconfig        types.String `tfsdk:"kubeconfig"`
	Host              types.String `tfsdk:"host"`
	CACertificate     types.String `tfsdk:"ca_certificate"`
	ClientCertificate types.String `tfsdk:"client_certificate"`
	ClientKey         types.String `tfsdk:"client_key"`
}

func (p *OLMProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "olm"
	resp.Version = p.version
}

func (p *OLMProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"kubeconfig": schema.StringAttribute{
				MarkdownDescription: "Kubeconfig raw file",
				Optional:            true,
				Sensitive:           true,
			},
			//"kubeconfig_path": schema.StringAttribute{
			//	MarkdownDescription: "path to kubeconfig file",
			//	Optional:            true,
			//},
			"host": schema.StringAttribute{
				MarkdownDescription: "Kubernetes API server host",
				Optional:            true,
			},
			"ca_certificate": schema.StringAttribute{
				MarkdownDescription: "Kubernetes API server CA certificate",
				Optional:            true,
			},
			"client_certificate": schema.StringAttribute{
				MarkdownDescription: "Kubernetes API server client certificate",
				Optional:            true,
			},
			"client_key": schema.StringAttribute{
				MarkdownDescription: "Kubernetes API server client key",
				Optional:            true,
			},
		},
	}
}

func (p *OLMProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data OLMProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Store the version and configuration for later client initialization
	p.config = &data
	resp.ResourceData = p
	resp.DataSourceData = p

}

func (p *OLMProvider) getClient() (*installer.Client, error) {

	// Return the client if it's already been initialized
	if p.client != nil {
		return p.client, nil
	}

	var config *rest.Config
	var err error

	// Check if the configuration is sufficient to initialize the client
	//if p.config.Kubeconfig.IsUnknown() || p.config.Kubeconfig.IsNull() {
	//	return nil, fmt.Errorf("insufficient configuration for OLM client initialization")
	//} else if p.config.Host.IsUnknown() || p.config.Host.IsNull() ||
	//	p.config.CACertificate.IsUnknown() || p.config.CACertificate.IsNull() ||
	//	p.config.ClientCertificate.IsUnknown() || p.config.ClientCertificate.IsNull() ||
	//	p.config.ClientKey.IsUnknown() || p.config.ClientKey.IsNull() {
	//
	//	return nil, fmt.Errorf("insufficient configuration for OLM client initialization")
	//
	//}
	// Check if the configuration is sufficient load the config into the config struct
	if !p.config.Kubeconfig.IsUnknown() && !p.config.Kubeconfig.IsNull() {

		// Use the raw kubeconfig string to build the config
		kubeconfigBytes := []byte(p.config.Kubeconfig.ValueString())
		config, err = clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
		if err != nil {
			return nil, err
		}
	} else if !p.config.Host.IsUnknown() || !p.config.Host.IsNull() ||
		!p.config.CACertificate.IsUnknown() || !p.config.CACertificate.IsNull() ||
		!p.config.ClientCertificate.IsUnknown() || !p.config.ClientCertificate.IsNull() ||
		!p.config.ClientKey.IsUnknown() || !p.config.ClientKey.IsNull() {
		// Use the host, ca_certificate, client_certificate and client_key to create the client
		config = &rest.Config{
			Host: p.config.Host.ValueString(),
			TLSClientConfig: rest.TLSClientConfig{
				CertData: []byte(p.config.ClientCertificate.ValueString()),
				KeyData:  []byte(p.config.ClientKey.ValueString()),
				CAData:   []byte(p.config.CACertificate.ValueString()),
			},
		}
	} else {
		return nil, fmt.Errorf("fucked up: insufficient configuration for OLM client initialization")

	}

	// Initialize the client
	p.client, err = installer.ClientForConfig(config)
	if err != nil {
		return nil, err
	}

	return p.client, err
}

func (p *OLMProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOLMv0Resource,
		NewOperatorv0Resource,
	}
}

func (p *OLMProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
