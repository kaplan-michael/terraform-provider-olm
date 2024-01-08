package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kaplan-michael/terraform-provider-olm/internal/olm/installer"
	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"strings"
)

// Ensure provider defined interface is implemented.
var _ resource.Resource = &Operatorv0Resource{}

// Operatorv0Resource struct.
type Operatorv0Resource struct {
	client *installer.Client // olm client
}

// NewOperatorv0Resource instantiates the resource with the Kubernetes client.
func NewOperatorv0Resource() resource.Resource {
	return &Operatorv0Resource{}
}

// Operatorv0ResourceModel represents the structure of the resource data.
type Operatorv0ResourceModel struct {
	Name                types.String `tfsdk:"name"`
	Channel             types.String `tfsdk:"channel"`
	Source              types.String `tfsdk:"source"`
	SourceNamespace     types.String `tfsdk:"source_namespace"`
	InstallPlanApproval types.String `tfsdk:"install_plan_approval"`
	Namespace           types.String `tfsdk:"namespace"`
	ID                  types.String `tfsdk:"id"`
}

func (r *Operatorv0Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_v0_operator"
}

// Schema returns the schema for the Operator v0 resource.
func (r *Operatorv0Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resource for managing Kubernetes namespaces",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Operator",
				Required:            true,
			},
			"channel": schema.StringAttribute{
				MarkdownDescription: "The update channel to use for the Operator",
				Required:            true,
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "The source catalog of the Operator",
				Optional:            true,
				Default:             stringdefault.StaticString("operatorhubio-catalog"),
				Computed:            true,
			},
			"source_namespace": schema.StringAttribute{
				MarkdownDescription: "The namespace where the Operator source catalog is installed",
				Optional:            true,
				Default:             stringdefault.StaticString("olm"),
				Computed:            true,
			},
			"install_plan_approval": schema.StringAttribute{
				MarkdownDescription: "The update approval strategy for the Operator install default is Automatic" +
					"Valid values are Automatic, Manual, but if you set Manual, the provider will not be able to install",
				Optional: true,
				Default:  stringdefault.StaticString(string(olmapiv1alpha1.ApprovalAutomatic)),
				Computed: true,
			},

			"namespace": schema.StringAttribute{
				MarkdownDescription: "The namespace where to install the Operator",
				Optional:            true,
				Default:             stringdefault.StaticString("operators"),
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Operator",
				Computed:            true,
			},
		},
	}
}

func (r *Operatorv0Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create method for Operatorv0Resource.
func (r *Operatorv0Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Operatorv0ResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the subscription resources
	resources, err := r.client.GetSubscriptionResources(
		plan.Name.ValueString(),
		plan.Namespace.ValueString(),
		plan.Channel.ValueString(),
		plan.Name.ValueString(),
		plan.Source.ValueString(),
		plan.SourceNamespace.ValueString(),
		plan.InstallPlanApproval.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Operator resources",
			fmt.Sprintf("Failed to get Operator resources: %v", err),
		)
		return
	}

	// Create the Operator
	operatorStatus, err := r.client.InstallOperator(ctx, resources)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to install Operator",
			fmt.Sprintf("Failed to install Operator: %v", err),
		)
		return
	}

	installed, err := operatorStatus.HasInstalledResources()
	if err != nil || !installed {
		resp.Diagnostics.AddError("Operator installation verification failed",
			"Operator resources are not installed as expected")
		return
	}

	// Set resource ID and state on successful creation
	id := plan.Name
	resp.State.Set(ctx, &Operatorv0ResourceModel{
		Name:                plan.Name,
		Channel:             plan.Channel,
		Source:              plan.Source,
		SourceNamespace:     plan.SourceNamespace,
		InstallPlanApproval: plan.InstallPlanApproval,
		Namespace:           plan.Namespace,
		ID:                  id,
	})
}

func (r *Operatorv0Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Operatorv0ResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the subscription resources
	resources, err := r.client.GetSubscriptionResources(
		state.Name.ValueString(),
		state.Namespace.ValueString(),
		state.Channel.ValueString(),
		state.Name.ValueString(),
		state.Source.ValueString(),
		state.SourceNamespace.ValueString(),
		state.InstallPlanApproval.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Operator resources",
			fmt.Sprintf("Failed to get Operator resources: %v", err),
		)
		return
	}

	// Get the current status
	status, err := r.client.GetSubscriptionStatus(ctx, resources)
	if err != nil {
		// The resource is not found, which we can assume is because it was deleted.
		// Remove the resource from the state and return.
		if strings.Contains(err.Error(), "the server could not find the requested resource") ||
			strings.Contains(err.Error(), "Can't find CSV for subscription") {
			resp.State.RemoveResource(ctx)
			return
		}
		// For other errors, report them back to Terraform.
		resp.Diagnostics.AddError("Error reading Operator status", err.Error())
		return
	}

	// Use the HasInstalledResources method to determine if the resources are installed
	installed, err := status.HasInstalledResources()
	if err != nil {
		resp.Diagnostics.AddError("Error checking Operator installation status", err.Error())
		return
	}

	if !installed {
		// If Operator is not installed, remove the resource from the state
		resp.State.RemoveResource(ctx)
		return
	}
	// Update the state - resources are present
	resp.State.Set(ctx, &state)
}

func (r *Operatorv0Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	//NoOp as we only support Automatic install plan approval, so there is nothing to update
}

func (r *Operatorv0Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Operatorv0ResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the subscription resources
	resources, err := r.client.GetSubscriptionResources(
		state.Name.ValueString(),
		state.Namespace.ValueString(),
		state.Channel.ValueString(),
		state.Name.ValueString(),
		state.Source.ValueString(),
		state.SourceNamespace.ValueString(),
		state.InstallPlanApproval.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Operator resources",
			fmt.Sprintf("Failed to get Operator resources: %v", err),
		)
		return
	}

	// Delete Operator using OLM client
	err = r.client.UninstallOperator(ctx, resources)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete Operator", err.Error())
		return
	}

	// Get the current status to verify deletion
	_, err = r.client.GetSubscriptionStatus(ctx, resources)
	if err != nil {
		// The resource is already deleted/not found, which is the desired outcome.
		// Remove the resource from the state and return.
		if strings.Contains(err.Error(), "the Operator is not installed") ||
			strings.Contains(err.Error(), "Can't find CSV for subscription") {
			resp.State.RemoveResource(ctx)
			return

		}
		// For other errors, report them back to Terraform.
		resp.Diagnostics.AddError("Error getting Operator status", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}
