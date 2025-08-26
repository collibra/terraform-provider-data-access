package utils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func StringArray[T ~string](values []T) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = string(value)
	}

	return result
}

func StringSetToSlice(ctx context.Context, set types.Set) (_ []string, diagnostics diag.Diagnostics) {
	var stringTypes []types.String
	diagnostics.Append(set.ElementsAs(ctx, &stringTypes, false)...)

	if diagnostics.HasError() {
		return nil, diagnostics
	}

	result := make([]string, len(stringTypes))
	for i, value := range stringTypes {
		result[i] = value.ValueString()
	}

	return result, diagnostics
}

func SliceToStringSet(_ context.Context, values []string) (types.Set, diag.Diagnostics) {
	stringTypes := make([]attr.Value, len(values))
	for i, value := range values {
		stringTypes[i] = types.StringValue(value)
	}

	return types.SetValue(types.StringType, stringTypes)
}

func Map[E ~[]S, S any, O any](values E, fn func(i S) O) []O {
	result := make([]O, len(values))

	for i := range values {
		result[i] = fn(values[i])
	}

	return result
}
