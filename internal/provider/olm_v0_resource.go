package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kaplan-michael/terraform-provider-olm/internal/olm/installer"
	"strings"
)

// Ensure provider defined interface is implemented.
var _ resource.Resource = &OLMv0Resource{}

// OLMv0Resource struct.
type OLMv0Resource struct {
	client *installer.Client // olm client
}

// NewOLMv0Resource instantiates the resource with the Kubernetes client.
func NewOLMv0Resource() resource.Resource {
	return &OLMv0Resource{}
}

// OlmV0ResourceModel represents the structure of the resource data.
type Olmv0ResourceModel struct {
	Namespace types.String `tfsdk:"namespace"`
	Version   types.String `tfsdk:"version"`
	ID        types.String `tfsdk:"id"`
}

func (r *OLMv0Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_v0_instance"
}

// Schema returns the schema for the OLM v0 resource.
func (r *OLMv0Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resource for managing Kubernetes namespaces",

		Attributes: map[string]schema.Attribute{
			"namespace": schema.StringAttribute{
				MarkdownDescription: "The namespace where to install olm",
				Optional:            true,
				Default:             stringdefault.StaticString("olm"),
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "OLM version to install v0 only",
				Optional:            true,
				Default:             stringdefault.StaticString(OLMv0Version),
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the OLM resource",
				Computed:            true,
			},
		},
	}
}

func (r *OLMv0Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*installer.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *installer.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client

}

// Create method for OLMv0Resource.
func (r *OLMv0Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Olmv0ResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	olmStatus, err := r.client.InstallVersion(ctx, plan.Namespace.ValueString(), plan.Version.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to install OLM",
			fmt.Sprintf("Failed to install OLM: %v", err),
		)
		return
	}

	installed, err := olmStatus.HasInstalledResources()
	if err != nil || !installed {
		resp.Diagnostics.AddError("OLM installation verification failed",
			"OLM resources are not installed as expected")
		return
	}

	// Set resource ID and state on successful creation
	id := "olm"
	resp.State.Set(ctx, &Olmv0ResourceModel{
		Namespace: plan.Namespace,
		Version:   plan.Version,
		ID:        types.StringValue(id),
	})
}

func (r *OLMv0Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Olmv0ResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Get the current status
	status, err := r.client.GetStatus(ctx, state.Version.ValueString())
	if err != nil {
		// The resource is not found, which we can assume is because it was deleted.
		// Remove the resource from the state and return.
		if strings.Contains(err.Error(), "the server could not find the requested resource") {
			resp.State.RemoveResource(ctx)
			return
		}
		// For other errors, report them back to Terraform.
		resp.Diagnostics.AddError("Error reading OLM status", err.Error())
		return
	}

	// Use the HasInstalledResources method to determine if the resources are installed
	installed, err := status.HasInstalledResources()
	if err != nil {
		resp.Diagnostics.AddError("Error checking OLM installation status", err.Error())
		return
	}

	if !installed {
		// If OLM is not installed, remove the resource from the state
		resp.State.RemoveResource(ctx)
		return
	}
	// Update the state - resources are present
	resp.State.Set(ctx, &state)
}

func (r *OLMv0Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan Olmv0ResourceModel
	var state Olmv0ResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if the version has changed
	if plan.Version != state.Version {
		// Uninstall the current version
		err := r.client.UninstallVersion(ctx, state.Version.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to uninstall the current OLM version", err.Error())
			return
		}
		// Install the new version
		olmStatus, err := r.client.InstallVersion(ctx, plan.Namespace.ValueString(), plan.Version.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to install the new OLM version", err.Error())
			return
		}

		installed, err := olmStatus.HasInstalledResources()
		if err != nil || !installed {
			resp.Diagnostics.AddError("OLM installation verification failed",
				"Failed to install the new OLM version. OLM resources are not installed as expected")
			return
		}

		// Update the state with the new version
		state.Version = plan.Version
	}

	// Update the Terraform state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *OLMv0Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Olmv0ResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete OLM using OLM client
	err := r.client.UninstallVersion(ctx, state.Version.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete OLM", err.Error())
		return
	}

	// Get the current status to verify deletion
	_, err = r.client.GetStatus(ctx, state.Version.ValueString())
	if err != nil {
		// The resource is already deleted/not found, which is the desired outcome.
		// Remove the resource from the state and return.
		if strings.Contains(err.Error(), "no existing installation found") {
			resp.State.RemoveResource(ctx)
			return

		}
		// For other errors, report them back to Terraform.
		resp.Diagnostics.AddError("Error getting OLM status", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}
