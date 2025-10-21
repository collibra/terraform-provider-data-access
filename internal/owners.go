package internal

import (
	"context"
	"fmt"

	"github.com/collibra/access-governance-go-sdk"
	"github.com/collibra/access-governance-go-sdk/services"
	dataAccessType "github.com/collibra/access-governance-go-sdk/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/collibra/access-governance-terraform-provider/internal/utils"
)

func getOwners(ctx context.Context, id string, client *sdk.CollibraClient) (result types.Set, diagnostics diag.Diagnostics) {
	ownersSeq := client.Role().ListRoleAssignments(ctx, services.WithRoleAssignmentListFilter(
		&dataAccessType.RoleAssignmentFilterInput{
			Role:               utils.Ptr(ownerRole),
			Resource:           &id,
			ExcludeDelegated:   utils.Ptr(true),
			ExcludeDelegations: utils.Ptr(true),
			Inherited:          utils.Ptr(false),
		},
	),
	)

	var owners []attr.Value

	for owner, err := range ownersSeq {
		if err != nil {
			diagnostics.AddError("Failed to list owners", err.Error())

			return result, diagnostics
		}

		switch ownerItem := owner.GetTo().(type) {
		case *dataAccessType.RoleAssignmentToUser:
			owners = append(owners, types.StringValue(ownerItem.Id))
		case *dataAccessType.RoleAssignmentToGroup:
			owners = append(owners, types.StringValue(ownerItem.Id))
		default:
			diagnostics.AddError("Unexpected owner type", fmt.Sprintf("Expected *types2.RoleAssignmentToUser or *types2.RoleAssignmentToGroup, got: %T. Please report this issue to the provider developers.", ownerItem))

			return result, diagnostics
		}
	}

	return types.SetValue(types.StringType, owners)
}
