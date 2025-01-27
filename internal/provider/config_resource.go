package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TGNThump/terraform-provider-vyos/internal/vyos"
	"github.com/foltik/vyos-client-go/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &ConfigResource{}
var _ resource.ResourceWithImportState = &ConfigResource{}
var _ resource.ResourceWithConfigure = &ConfigResource{}

func NewConfigResource() resource.Resource {
	return &ConfigResource{}
}

// ConfigResource defines the resource implementation.
type ConfigResource struct {
	vyosConfig *vyos.VyosConfig
}

// ConfigResourceModel describes the resource data model.
type ConfigResourceModel struct {
	Path  types.String `tfsdk:"path"`
	Value types.String `tfsdk:"value"`
	Id    types.String `tfsdk:"id"`
}

func (r *ConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

func (r *ConfigResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Configuration Resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Configuration identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "Configuration path",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "JSON configuration for the path",
				Optional:            true,
			},
		},
	}
}

func (r *ConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ConfigResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check if config already exists
	tflog.Info(ctx, "Reading path "+data.Path.ValueString())

	components := strings.Split(data.Path.ValueString(), " ")
	parentPath := strings.Join(components[0:len(components)-1], " ")
	terminal := components[len(components)-1]

	parent, err := r.vyosConfig.Show(ctx, parentPath)
	if err != nil {
		resp.Diagnostics.AddError("No", err.Error())
		return
	}

	if parent != nil {
		existing := parent.(map[string]any)[terminal]

		if existing != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Configuration path '%s' already exists, try a resource import instead.", data.Path.ValueString()), fmt.Sprintf("%v", existing))
			return
		}
	}

	var jsonValue interface{}
	err = json.Unmarshal([]byte(data.Value.ValueString()), &jsonValue)

	if err != nil {
		resp.Diagnostics.AddError("No", err.Error())
		return
	}

	tflog.Info(ctx, "Setting path "+data.Path.ValueString()+" to value "+data.Value.ValueString())

	err = r.vyosConfig.Set(ctx, data.Path.ValueString(), jsonValue)
	if err != nil {
		resp.Diagnostics.AddError("No", err.Error())
		return
	}

	data.Id = types.StringValue(data.Path.ValueString())

	tflog.Info(ctx, "Set path "+data.Path.ValueString()+" to value "+data.Value.ValueString())

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ConfigResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading path "+data.Path.ValueString())

	components := strings.Split(data.Path.ValueString(), " ")
	parentPath := strings.Join(components[0:len(components)-1], " ")
	terminal := components[len(components)-1]

	parent, err := r.vyosConfig.Show(ctx, parentPath)
	if err != nil {
		resp.Diagnostics.AddError("No", err.Error())
		return
	}

	if parent == nil {
		resp.Diagnostics.AddError("Parent of resource not found", "Parent not found")
		return
	}

	config := parent.(map[string]any)[terminal]

	if config == nil {
		resp.Diagnostics.AddError("Resource not found", "Parent not found")
		return
	}

	jsonValue, err := json.Marshal(config)
	if err != nil {
		resp.Diagnostics.AddError("No", err.Error())
		return
	}

	data.Value = types.StringValue(string(jsonValue[:]))

	tflog.Info(ctx, "Read path "+data.Path.ValueString()+" with value "+data.Value.ValueString())

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan *ConfigResourceModel
	var state *ConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating path "+plan.Path.ValueString()+" to value "+plan.Value.ValueString())

	var payload []map[string]any

	payload = append(payload, map[string]any{
		"op":   "delete",
		"path": strings.Split(plan.Path.ValueString(), " "),
	})

	{
		var value interface{}
		err := json.Unmarshal([]byte(plan.Value.ValueString()), &value)

		if err != nil {
			resp.Diagnostics.AddError("No", err.Error())
			return
		}

		flat, err := client.Flatten(value)
		if err != nil {
			resp.Diagnostics.AddError("No", err.Error())
			return
		}

		for _, pair := range flat {
			subpath, value := pair[0], pair[1]

			prefixpath := plan.Path.ValueString()
			if len(prefixpath) > 0 && len(subpath) > 0 {
				prefixpath += " "
			}
			prefixpath += subpath

			payload = append(payload, map[string]any{
				"op":    "set",
				"path":  strings.Split(prefixpath, " "),
				"value": value,
			})
		}
	}

	tflog.Info(ctx, fmt.Sprintf("%v", payload))

	_, err := r.vyosConfig.ApiRequest(ctx, "configure", payload)
	if err != nil {
		resp.Diagnostics.AddError("No", err.Error())
		return
	}

	tflog.Info(ctx, "Updated path "+plan.Path.ValueString()+" to value "+plan.Value.ValueString())

	// Save updated plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ConfigResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting path "+data.Path.ValueString())

	err := r.vyosConfig.Delete(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("No", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted path "+data.Path.ValueString())
}

func (r *ConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)
}
