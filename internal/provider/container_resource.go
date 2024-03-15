package provider

import (
	"context"
	"fmt"

	"github.com/TGNThump/terraform-provider-vyos/internal/vyos"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &ContainerResource{}
var _ resource.ResourceWithImportState = &ContainerResource{}
var _ resource.ResourceWithConfigure = &ContainerResource{}

func NewContainerResource() resource.Resource {
	return &ContainerResource{}
}

// ContainerResource defines the resource implementation.
type ContainerResource struct {
	vyosConfig *vyos.VyosConfig
}

// ContainerResourceModel describes the resource data model.
type ContainerResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Image       types.String `tfsdk:"image"`
	Description types.String `tfsdk:"description"`
	HostNetwork types.Bool   `tfsdk:"host_network"`
	Network     types.Object `tfsdk:"network"`
	Env         types.Map    `tfsdk:"env"`
	Ports       types.List   `tfsdk:"ports"`
}

type ContainerNetworkResourceModel struct {
	Name    types.String `tfsdk:"name"`
	Address types.String `tfsdk:"address"`
}

type ContainerPortResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Source      types.Number `tfsdk:"source"`
	Destination types.Number `tfsdk:"destination"`
	Protocol    types.String `tfsdk:"protocol"`
}

type ContainerVolumeResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Source      types.Number `tfsdk:"source"`
	Destination types.Number `tfsdk:"destination"`
	Mode        types.String `tfsdk:"mode"`
}

func (r *ContainerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

func (r *ContainerResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Container Resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Configuration identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Container name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "Container image name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Container description",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"host_network": schema.BoolAttribute{
				MarkdownDescription: "Whether to use host network. Mutually exclusive with 'network'",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"network": schema.ObjectAttribute{
				MarkdownDescription: "Container network",
				Optional:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				AttributeTypes: map[string]attr.Type{
					"name":    types.StringType,
					"address": types.StringType,
				},
			},
			"env": schema.MapAttribute{
				MarkdownDescription: "Container environment variables",
				Optional:            true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				ElementType: types.StringType,
			},
			"ports": schema.ListAttribute{
				MarkdownDescription: "Container ports",
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name":        types.StringType,
						"source":      types.NumberType,
						"destination": types.NumberType,
						"protocol":    types.StringType,
					},
				},
			},
			"volumes": schema.ListAttribute{
				MarkdownDescription: "Container volumes",
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name":        types.StringType,
						"source":      types.NumberType,
						"destination": types.NumberType,
						"mode":        types.StringType,
					},
				},
			},
		},
	}
}

func (r *ContainerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	vyosConfig, ok := req.ProviderData.(*vyos.VyosConfig)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.vyosConfig = vyosConfig
}

func (r *ContainerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ContainerResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Adding container image "+data.Name.ValueString())

	err := r.vyosConfig.ContainerImages().Add(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("No", err.Error())
		return
	}
	data.Id = types.StringValue(data.Name.ValueString())

	tflog.Info(ctx, "Added container image "+data.Name.ValueString())

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ContainerResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Getting container image "+data.Name.ValueString())

	image, err := r.vyosConfig.ContainerImages().Show(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get container images", err.Error())
		return
	}

	if image == nil {
		// resp.Diagnostics.AddError(fmt.Sprintf("No image exists with name %s", "data.Name.ValueString()"), err.Error())
		return
	}

	data.Name = types.StringValue(fmt.Sprintf("%s:%s", image.Name, image.Tag))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// var plan *ContainerResourceModel
	// var state *ContainerResourceModel

	// resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	// resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// if resp.Diagnostics.HasError() {
	// 	return
	// }

	// tflog.Info(ctx, "Updating path "+plan.Path.ValueString()+" to value "+plan.Value.ValueString())

	// var payload []map[string]any

	// payload = append(payload, map[string]any{
	// 	"op":   "delete",
	// 	"path": strings.Split(plan.Path.ValueString(), " "),
	// })

	// {
	// 	var value interface{}
	// 	err := json.Unmarshal([]byte(plan.Value.ValueString()), &value)

	// 	if err != nil {
	// 		resp.Diagnostics.AddError("No", err.Error())
	// 		return
	// 	}

	// 	flat, err := client.Flatten(value)
	// 	if err != nil {
	// 		resp.Diagnostics.AddError("No", err.Error())
	// 		return
	// 	}

	// 	for _, pair := range flat {
	// 		subpath, value := pair[0], pair[1]

	// 		prefixpath := plan.Path.ValueString()
	// 		if len(prefixpath) > 0 && len(subpath) > 0 {
	// 			prefixpath += " "
	// 		}
	// 		prefixpath += subpath

	// 		payload = append(payload, map[string]any{
	// 			"op":    "set",
	// 			"path":  strings.Split(prefixpath, " "),
	// 			"value": value,
	// 		})
	// 	}
	// }

	// tflog.Info(ctx, fmt.Sprintf("%v", payload))

	// _, err := r.vyosConfig.ApiRequest(ctx, "configure", payload)
	// if err != nil {
	// 	resp.Diagnostics.AddError("No", err.Error())
	// 	return
	// }

	// tflog.Info(ctx, "Updated path "+plan.Path.ValueString()+" to value "+plan.Value.ValueString())

	// // Save updated plan into Terraform state
	// resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ContainerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ContainerResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting container image "+data.Name.ValueString())

	err := r.vyosConfig.ContainerImages().Delete(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting container image", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted container image "+data.Name.ValueString())
}

func (r *ContainerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
