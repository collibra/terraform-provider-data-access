package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/golang-set/set"
	"github.com/raito-io/sdk-go"
	raitoType "github.com/raito-io/sdk-go/types"

	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*DataSourceResource)(nil)

type DataSourceResourceModel struct {
	Id                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	SyncMethod          types.String `tfsdk:"sync_method"`
	Parent              types.String `tfsdk:"parent"`
	NativeIdentityStore types.String `tfsdk:"native_identity_store"`
	IdentityStores      types.Set    `tfsdk:"identity_stores"`
	Owners              types.Set    `tfsdk:"owners"`
}

func (m *DataSourceResourceModel) ToDataSourceInput() raitoType.DataSourceInput {
	return raitoType.DataSourceInput{
		Name:        m.Name.ValueStringPointer(),
		Description: m.Description.ValueStringPointer(),
		SyncMethod:  utils.Ptr(raitoType.DataSourceSyncMethod(m.SyncMethod.ValueString())),
		Parent:      m.Parent.ValueStringPointer(),
	}
}

type DataSourceResource struct {
	client *sdk.RaitoClient
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
			"sync_method": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "The sync method of the data source (should be ON_PREM for now)",
				MarkdownDescription: "The sync method of the data source (should be `ON_PREM` for now)",
				Default:             stringdefault.StaticString(string(raitoType.DataSourceSyncMethodOnPrem)),
				Validators:          []validator.String{stringvalidator.OneOf(string(raitoType.DataSourceSyncMethodOnPrem), string(raitoType.DataSourceSyncMethodCloudManualTrigger))},
			},
			"parent": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            false,
				Sensitive:           false,
				Description:         "The ID of the parent data source, if applicable",
				MarkdownDescription: "The ID of the parent data source, if applicable",
			},
			"native_identity_store": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the native identity store",
				MarkdownDescription: "The ID of the native identity store",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"identity_stores": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "The IDs of the linked identity stores",
				MarkdownDescription: "The IDs of the linked identity stores",
				DeprecationMessage:  "",
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
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
		},
		Description:         "The data source resource",
		MarkdownDescription: "The resource for representing a Raito [Data Source](https://docs.raito.io/docs/cloud/datasources).",
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

	// Load current identity stores
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	identityStores, err := d.client.DataSource().ListIdentityStores(cancelCtx, dataSourceResult.Id)
	if err != nil {
		response.Diagnostics.AddError("Failed to list identity stores", err.Error())

		return
	}

	planExpectedIss := set.Set[string]{}

	for _, identityStoreId := range data.IdentityStores.Elements() {
		idValue := identityStoreId.(types.String)
		planExpectedIss.Add(idValue.ValueString())
	}

	linkedIss := set.Set[string]{}

	for _, identityStore := range identityStores {
		if identityStore.Native {
			data.NativeIdentityStore = types.StringValue(identityStore.Id)
		} else {
			linkedIss.Add(identityStore.Id)
		}
	}

	// Add missing identity stores
	for is := range planExpectedIss {
		if linkedIss.Contains(is) {
			continue
		}

		err = d.client.DataSource().AddIdentityStoreToDataSource(ctx, dataSourceResult.Id, is)
		if err != nil {
			response.Diagnostics.AddError("Failed to remove identity store from data source", err.Error())

			return
		}
	}

	// Remove old identity stores
	for is := range linkedIss {
		if planExpectedIss.Contains(is) {
			continue
		}

		err = d.client.DataSource().RemoveIdentityStoreFromDataSource(ctx, dataSourceResult.Id, is)
		if err != nil {
			response.Diagnostics.AddError("Failed to add identity store to data source", err.Error())

			return
		}
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

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
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
		var notFoundErr *raitoType.ErrNotFound
		if errors.As(err, &notFoundErr) {
			response.State.RemoveResource(ctx)
		} else {
			response.Diagnostics.AddError("Failed to get data source", err.Error())
		}

		return
	}

	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	identityStores, err := d.client.DataSource().ListIdentityStores(cancelCtx, stateData.Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Failed to list identity stores", err.Error())

		return
	}

	var nativeIs *string
	isIds := make([]attr.Value, 0, len(identityStores))

	for i, identityStore := range identityStores {
		if identityStore.Native {
			nativeIs = &identityStores[i].Id
		} else if !identityStore.Master {
			isIds = append(isIds, types.StringValue(identityStore.Id))
		}
	}

	isAttr, diagnostic := types.SetValue(types.StringType, isIds)
	response.Diagnostics.Append(diagnostic...)

	var parentId *string
	if ds.Parent != nil {
		parentId = &ds.Parent.Id
	}

	if response.Diagnostics.HasError() {
		return
	}

	actualData := DataSourceResourceModel{
		Id:                  types.StringValue(ds.Id),
		Name:                types.StringValue(ds.Name),
		Description:         types.StringValue(ds.Description),
		SyncMethod:          types.StringValue(string(ds.SyncMethod)),
		Parent:              types.StringPointerValue(parentId),
		NativeIdentityStore: types.StringPointerValue(nativeIs),
		IdentityStores:      isAttr,
	}

	owners, diagn := getOwners(ctx, stateData.Id.ValueString(), d.client)
	response.Diagnostics.Append(diagn...)

	if response.Diagnostics.HasError() {
		return
	}

	actualData.Owners = owners

	response.Diagnostics.Append(response.State.Set(ctx, actualData)...)
}

func (d *DataSourceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data DataSourceResourceModel

	// Read Terraform plan data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Update data source
	_, err := d.client.DataSource().UpdateDataSource(ctx, data.Id.ValueString(), data.ToDataSourceInput())
	if err != nil {
		response.Diagnostics.AddError("Failed to update data source", err.Error())

		return
	}

	// Load current identity stores
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	identityStores, err := d.client.DataSource().ListIdentityStores(cancelCtx, data.Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Failed to list identity stores", err.Error())

		return
	}

	planExpectedIss := set.Set[string]{}

	for _, identityStoreId := range data.IdentityStores.Elements() {
		idValue := identityStoreId.(types.String)
		planExpectedIss.Add(idValue.ValueString())
	}

	linkedIss := set.Set[string]{}

	for _, identityStore := range identityStores {
		if !identityStore.Native && !identityStore.Master {
			linkedIss.Add(identityStore.Id)
		}
	}

	// Add missing identity stores
	for is := range planExpectedIss {
		if linkedIss.Contains(is) {
			continue
		}

		err = d.client.DataSource().AddIdentityStoreToDataSource(ctx, data.Id.ValueString(), is)
		if err != nil {
			response.Diagnostics.AddError("Failed to remove identity store from data source", err.Error())

			return
		}
	}

	// Remove old identity stores
	for is := range linkedIss {
		if planExpectedIss.Contains(is) {
			continue
		}

		err = d.client.DataSource().RemoveIdentityStoreFromDataSource(ctx, data.Id.ValueString(), is)
		if err != nil {
			response.Diagnostics.AddError("Failed to add identity store to data source", err.Error())

			return
		}
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

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
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

	client, ok := req.ProviderData.(*sdk.RaitoClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sdk.RaitoClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	if client == nil {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *sdk.RaitoClient, not to be nil.",
		)

		return
	}

	d.client = client
}

func (d *DataSourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
