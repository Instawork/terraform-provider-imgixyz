package internal

// Pulled example from
// https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#creating-attribute-plan-modifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

func UseStateAfterSetModifier() planmodifier.String {
	return useStateAfterSetModifier{}
}

// useStateAfterSetModifier implements the plan modifier.
type useStateAfterSetModifier struct{}

// Description returns a human-readable description of the plan modifier.
func (m useStateAfterSetModifier) Description(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m useStateAfterSetModifier) MarkdownDescription(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

// PlanModifyBool implements the plan modification logic.
func (m useStateAfterSetModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Do nothing if there is no state value.
	if req.StateValue.IsNull() {
		return
	}

	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() {
		return
	}

	resp.PlanValue = req.StateValue
}
