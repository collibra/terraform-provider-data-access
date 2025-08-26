package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk-go"
	"github.com/raito-io/sdk-go/services"
	raitoType "github.com/raito-io/sdk-go/types"

	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

func getOwners(ctx context.Context, id string, client *sdk.RaitoClient) (result types.Set, diagnostics diag.Diagnostics) {
	ownersList := client.Role().ListRoleAssignments(ctx, services.WithRoleAssignmentListFilter(
		&raitoType.RoleAssignmentFilterInput{
			Role:               utils.Ptr(ownerRole),
			Resource:           &id,
			ExcludeDelegated:   utils.Ptr(true),
			ExcludeDelegations: utils.Ptr(true),
			Inherited:          utils.Ptr(false),
		},
	),
	)

	var owners []attr.Value

	for owner := range ownersList {
		if owner.HasError() {
			diagnostics.AddError("Failed to list owners", owner.GetError().Error())

			return result, diagnostics
		}

		switch ownerItem := owner.GetItem().GetTo().(type) {
		case *raitoType.RoleAssignmentToUser:
			owners = append(owners, types.StringValue(ownerItem.Id))
		case *raitoType.RoleAssignmentToGroup:
			owners = append(owners, types.StringValue(ownerItem.Id))
		default:
			diagnostics.AddError("Unexpected owner type", fmt.Sprintf("Expected *types2.RoleAssignmentToUser or *types2.RoleAssignmentToGroup, got: %T. Please report this issue to the provider developers.", ownerItem))

			return result, diagnostics
		}
	}

	return types.SetValue(types.StringType, owners)
}
