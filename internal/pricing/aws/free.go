package aws

import (
	"context"

	"github.com/osifotos/price-checker/internal/pricing"
	"github.com/osifotos/price-checker/pkg/schema"
)

// ---- resource types with no direct AWS charge of their own ----

// freeResourcePricer covers resource types AWS never bills for directly —
// their cost, if any, is attributed entirely to other resources (e.g. the EC2
// instances a security group is attached to) or to usage this tool does not
// model (e.g. per-request API Gateway charges). $0.00 is the correct, final
// answer for these, so they are reported as NOT_BILLABLE rather than
// UNSUPPORTED_TYPE.
type freeResourcePricer struct{}

func (freeResourcePricer) ResourceTypes() []string {
	return []string{
		"aws_route_table",
		"aws_route_table_association",
		"aws_security_group",
		"aws_security_group_rule",
		"aws_subnet",
		"aws_vpc",
		"aws_internet_gateway",
		"aws_iam_role",
		"aws_iam_policy",
		"aws_iam_role_policy_attachment",
		"aws_iam_instance_profile",
		"aws_ecs_cluster",
		"aws_apigatewayv2_authorizer",
		"aws_apigatewayv2_stage",
	}
}

func (freeResourcePricer) Price(_ context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	out.Skip(in.Resource, "", schema.ReasonNotBillable)
	return out, nil
}
