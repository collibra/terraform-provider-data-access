package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/collibra/data-access-go-sdk"
	dataAccessType "github.com/collibra/data-access-go-sdk/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*DataSourceResource)(nil)

type DataSourceResourceModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Type             types.String `tfsdk:"type"`
	Parent           types.String `tfsdk:"parent"`
	Owners           types.Set    `tfsdk:"owners"`
	EdgeSiteId       types.String `tfsdk:"edge_site_id"`
	EdgeConnectionId types.String `tfsdk:"edge_connection_id"`
	SyncParameters   types.Map    `tfsdk:"sync_parameters"`
}

func (m *DataSourceResourceModel) ToDataSourceInput() dataAccessType.DataSourceInput {
	return dataAccessType.DataSourceInput{
		Name:             m.Name.ValueStringPointer(),
		Description:      m.Description.ValueStringPointer(),
		Type:             m.Type.ValueStringPointer(),
		Parent:           m.Parent.ValueStringPointer(),
		EdgeSiteId:       m.EdgeSiteId.ValueStringPointer(),
		EdgeConnectionId: m.EdgeConnectionId.ValueStringPointer(),
	}
}

type DataSourceResource struct {
	client *sdk.CollibraClient
}

func NewDataSourceResource() resource.Resource {
	return &DataSourceResource{}
}

func (d *DataSourceResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_datasource"
}

func (d *DataSourceResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the data source",
				MarkdownDescription: "The ID of the data source",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "The name of the data source",
				MarkdownDescription: "The name of the data source",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(3)},
			},
			"description": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "The description of the data source",
				MarkdownDescription: "The description of the data source",
				Default:             stringdefault.StaticString(""),
			},
			"type": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            false,
				Sensitive:           false,
				Description:         "The type of the data source (e.g. Snowflake, BigQuery). Required when edge_site_id or edge_connection_id is set.",
				MarkdownDescription: "The type of the data source (e.g. Snowflake, BigQuery). Required when `edge_site_id` or `edge_connection_id` is set.",
			},
			"parent": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            false,
				Sensitive:           false,
				Description:         "The ID of the parent data source, if applicable",
				MarkdownDescription: "The ID of the parent data source, if applicable",
			},
			"owners": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "The IDs of the owners of the data source",
				MarkdownDescription: "The IDs of the owners of the data source",
			},
			"edge_site_id": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            false,
				Sensitive:           false,
				Description:         "The ID of the Edge Site associated with this data source. Requires edge_connection_id and type to also be set.",
				MarkdownDescription: "The ID of the Edge Site associated with this data source. Requires `edge_connection_id` and `type` to also be set.",
				Validators:          []validator.String{stringvalidator.AlsoRequires(path.MatchRoot("type"), path.MatchRoot("edge_connection_id"))},
			},
			"edge_connection_id": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            false,
				Sensitive:           false,
				Description:         "The ID of the Edge Connection associated with this data source. Requires edge_site_id and type to also be set.",
				MarkdownDescription: "The ID of the Edge Connection associated with this data source. Requires `edge_site_id` and `type` to also be set.",
				Validators:          []validator.String{stringvalidator.AlsoRequires(path.MatchRoot("type"), path.MatchRoot("edge_site_id"))},
			},
			"sync_parameters": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            false,
				Description:         "Sync configuration parameters as a map of dot-notation paths to JSON-encoded values. E.g. {\"global.sf-tags\" = \"true\", \"global.page-size\" = \"42\", \"global.tag-name\" = \"\\\"myValue\\\"\"}. Set a key to null to remove it.",
				MarkdownDescription: "Sync configuration parameters as a map of dot-notation paths to JSON-encoded values.",
			},
		},
		Description:         "The data source resource",
		MarkdownDescription: "The resource for representing a Data Source.",
		Version:             1,
	}
}

func (d *DataSourceResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data DataSourceResourceModel

	// Read Terraform plan data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Create data source
	dataSourceResult, err := d.client.DataSource().CreateDataSource(ctx, data.ToDataSourceInput())
	if err != nil {
		response.Diagnostics.AddError("Failed to create data source", err.Error())

		return
	}

	data.Id = types.StringValue(dataSourceResult.Id)
	response.Diagnostics.Append(response.State.Set(ctx, data)...) //Ensure to store id first

	// Set sync parameters
	response.Diagnostics.Append(
		d.setSyncParameters(ctx, dataSourceResult.Id, types.MapNull(types.StringType), data.SyncParameters)...,
	)

	if response.Diagnostics.HasError() {
		return
	}

	// Set Owners
	if !data.Owners.IsNull() && len(data.Owners.Elements()) > 0 {
		response.Diagnostics.Append(d.setOwners(ctx, &data.Owners, dataSourceResult.Id)...)

		if response.Diagnostics.HasError() {
			return
		}
	}

	owners, diagn := getOwners(ctx, dataSourceResult.Id, d.client)
	response.Diagnostics.Append(diagn...)

	if response.Diagnostics.HasError() {
		return
	}

	data.Owners = owners

	hydratedData, diagn := d.readDataSourceState(ctx, dataSourceResult.Id, data.SyncParameters)
	response.Diagnostics.Append(diagn...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, hydratedData)...)
}

func (d *DataSourceResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var stateData DataSourceResourceModel

	// Read Terraform plan stateData into the model
	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)

	if response.Diagnostics.HasError() {
		return
	}

	ds, err := d.client.DataSource().GetDataSource(ctx, stateData.Id.ValueString())
	if err != nil {
		var notFoundErr *dataAccessType.ErrNotFound
		if errors.As(err, &notFoundErr) {
			response.State.RemoveResource(ctx)
		} else {
			response.Diagnostics.AddError("Failed to get data source", err.Error())
		}

		return
	}

	actualData, diagn := d.readDataSourceState(ctx, ds.Id, stateData.SyncParameters)
	response.Diagnostics.Append(diagn...)

	response.Diagnostics.Append(response.State.Set(ctx, actualData)...)
}

func (d *DataSourceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data DataSourceResourceModel

	// Read Terraform plan data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	var priorData DataSourceResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &priorData)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Update data source
	_, err := d.client.DataSource().UpdateDataSource(ctx, data.Id.ValueString(), data.ToDataSourceInput())
	if err != nil {
		response.Diagnostics.AddError("Failed to update data source", err.Error())

		return
	}

	// Set sync parameters
	response.Diagnostics.Append(
		d.setSyncParameters(ctx, data.Id.ValueString(), priorData.SyncParameters, data.SyncParameters)...,
	)

	if response.Diagnostics.HasError() {
		return
	}

	// Set Owners
	if !data.Owners.IsNull() && len(data.Owners.Elements()) > 0 {
		response.Diagnostics.Append(d.setOwners(ctx, &data.Owners, data.Id.ValueString())...)

		if response.Diagnostics.HasError() {
			return
		}
	}

	owners, diagn := getOwners(ctx, data.Id.ValueString(), d.client)
	response.Diagnostics.Append(diagn...)

	if response.Diagnostics.HasError() {
		return
	}

	data.Owners = owners

	hydratedData, diagn := d.readDataSourceState(ctx, data.Id.ValueString(), data.SyncParameters)
	response.Diagnostics.Append(diagn...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, hydratedData)...)
}

func (d *DataSourceResource) readDataSourceState(
	ctx context.Context,
	id string,
	syncParameters types.Map,
) (DataSourceResourceModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	ds, err := d.client.DataSource().GetDataSource(ctx, id)
	if err != nil {
		diagnostics.AddError("Failed to get data source", err.Error())

		return DataSourceResourceModel{}, diagnostics
	}

	var parentId *string
	if ds.Parent != nil {
		parentId = &ds.Parent.Id
	}

	var edgeSiteId, edgeConnectionId *string

	if ds.EdgeSiteInfo != nil {
		type edgeSiteInfoGetter interface {
			GetEdgeSiteId() *string
			GetEdgeConnectionId() *string
		}
		if info, ok := (*ds.EdgeSiteInfo).(edgeSiteInfoGetter); ok {
			edgeSiteId = info.GetEdgeSiteId()
			edgeConnectionId = info.GetEdgeConnectionId()
		}
	}

	owners, ownerDiagnostics := getOwners(ctx, ds.Id, d.client)
	diagnostics.Append(ownerDiagnostics...)

	if diagnostics.HasError() {
		return DataSourceResourceModel{}, diagnostics
	}

	return DataSourceResourceModel{
		Id:               types.StringValue(ds.Id),
		Name:             types.StringValue(ds.Name),
		Description:      types.StringValue(ds.Description),
		Type:             types.StringValue(ds.Type),
		Parent:           types.StringPointerValue(parentId),
		Owners:           owners,
		EdgeSiteId:       types.StringPointerValue(edgeSiteId),
		EdgeConnectionId: types.StringPointerValue(edgeConnectionId),
		SyncParameters:   syncParameters,
	}, diagnostics
}

func (d *DataSourceResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data DataSourceResourceModel

	// Read Terraform plan data into the model
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	currentUser, err := d.client.User().GetCurrentUser(ctx)
	if err != nil {
		response.Diagnostics.AddError("Failed to get current user", err.Error())

		return
	}

	_, err = d.client.Role().UpdateRoleAssigneesOnDataSource(ctx, data.Id.ValueString(), ownerRole, currentUser.Id)
	if err != nil {
		response.Diagnostics.AddError("Failed to remove role assignees from data source", err.Error())

		return
	}

	err = d.client.DataSource().DeleteDataSource(ctx, data.Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Failed to delete data source", err.Error())

		return
	}

	response.State.RemoveResource(ctx)
}

func (d *DataSourceResource) setSyncParameters(
	ctx context.Context,
	dsId string,
	priorParams, newParams types.Map,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if newParams.IsNull() && priorParams.IsNull() {
		return diags
	}

	var values []dataAccessType.SyncParameterValueInput

	// Build values to set/update
	if !newParams.IsNull() {
		newElements := map[string]types.String{}
		diags.Append(newParams.ElementsAs(ctx, &newElements, false)...)

		if diags.HasError() {
			return diags
		}

		for path, jsonStr := range newElements {
			var decoded interface{}
			if err := json.Unmarshal([]byte(jsonStr.ValueString()), &decoded); err != nil {
				diags.AddError(
					"Invalid sync parameter value",
					fmt.Sprintf("Value for path %q is not valid JSON: %s", path, err),
				)
				return diags
			}

			decodedVal := decoded
			values = append(values, dataAccessType.SyncParameterValueInput{Path: path, Value: &decodedVal})
		}
	}

	// Build removed keys (nil = delete from backend)
	if !priorParams.IsNull() {
		priorElements := map[string]types.String{}
		diags.Append(priorParams.ElementsAs(ctx, &priorElements, false)...)

		if diags.HasError() {
			return diags
		}

		newElements := map[string]types.String{}
		if !newParams.IsNull() {
			diags.Append(newParams.ElementsAs(ctx, &newElements, false)...)

			if diags.HasError() {
				return diags
			}
		}

		for path := range priorElements {
			if _, exists := newElements[path]; !exists {
				values = append(values, dataAccessType.SyncParameterValueInput{Path: path, Value: nil})
			}
		}
	}

	if len(values) == 0 {
		return diags
	}

	_, err := d.client.DataSource().SetSyncConfigurationParameterValues(
		ctx,
		dataAccessType.SyncParameterValuesInput{DataSourceId: dsId, Values: values},
	)
	if err != nil {
		diags.AddError("Failed to set sync configuration parameter values", err.Error())
	}

	return diags
}

func (d *DataSourceResource) setOwners(ctx context.Context, ownerSet *types.Set, dsId string) (diagnostics diag.Diagnostics) {
	ownersValues := ownerSet.Elements()
	owners := make([]string, 0, len(ownersValues))

	for _, owner := range ownersValues {
		owners = append(owners, owner.(types.String).ValueString())
	}

	_, err := d.client.Role().UpdateRoleAssigneesOnDataSource(ctx, dsId, ownerRole, owners...)
	if err != nil {
		diagnostics.AddError("Failed to update role assignees on data source", err.Error())
	}

	return diagnostics
}

func (d *DataSourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.CollibraClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sdk.CollibraClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	if client == nil {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *sdk.CollibraClient, not to be nil.",
		)

		return
	}

	d.client = client
}

func (d *DataSourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
