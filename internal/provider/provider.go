// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
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

	// Check if at least one value is set.
	if data.Kubeconfig.IsUnknown() || data.Kubeconfig.IsNull() {
		if data.Host.IsUnknown() || data.Host.IsNull() ||
			data.CACertificate.IsUnknown() || data.CACertificate.IsNull() ||
			data.ClientCertificate.IsUnknown() || data.ClientCertificate.IsNull() ||
			data.ClientKey.IsUnknown() || data.ClientKey.IsNull() {
			resp.Diagnostics.AddError(
				"Configuration Error",
				"Either the `kubeconfig` must be configured, or `host`, `ca_certificate`,"+
					" `client_certificate` and `client_key` must be configured for the provider",
			)
		}
		return
	}
	var client *installer.Client
	var err error

	if !data.Kubeconfig.IsUnknown() && !data.Kubeconfig.IsNull() {

		// Use the raw kubeconfig string to build the config
		kubeconfigBytes := []byte(data.Kubeconfig.ValueString())
		config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
		if err != nil {
			// Handle error
			return
		}
		client, err = installer.ClientForConfig(config)
		if err != nil {
			// Handle error
			return
		}
	} else {
		// Use the host, ca_certificate, client_certificate and client_key to create the client
		config := &rest.Config{
			Host: data.Host.ValueString(),
			TLSClientConfig: rest.TLSClientConfig{
				CertData: []byte(data.ClientCertificate.ValueString()),
				KeyData:  []byte(data.ClientKey.ValueString()),
				CAData:   []byte(data.CACertificate.ValueString()),
			},
		}
		client, err = installer.ClientForConfig(config)
		if err != nil {
			// Handle err
			return
		}

	}

	// Provide the client to resources and data sources.
	resp.ResourceData = client
	resp.DataSourceData = client
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
