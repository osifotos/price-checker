package schema

// ReasonCode is the closed set of machine-readable explanations for why a
// resource or cost component could not be estimated.
type ReasonCode string

const (
	// ReasonUnsupportedType: no pricer in the catalog handles this resource type.
	ReasonUnsupportedType ReasonCode = "UNSUPPORTED_TYPE"
	// ReasonMissingAttribute: a required attribute was absent from the plan.
	ReasonMissingAttribute ReasonCode = "MISSING_ATTRIBUTE"
	// ReasonUnknownAfterApply: a required attribute is "known after apply".
	ReasonUnknownAfterApply ReasonCode = "UNKNOWN_AFTER_APPLY"
	// ReasonNoUsageData: a usage-based component has no value in the usage file.
	ReasonNoUsageData ReasonCode = "NO_USAGE_DATA"
	// ReasonPricingAPIError: the AWS Price List API could not be queried.
	ReasonPricingAPIError ReasonCode = "PRICING_API_ERROR"
	// ReasonUnsupportedConfiguration: the resource type is supported but this
	// particular configuration is not (for example an unmodelled engine).
	ReasonUnsupportedConfiguration ReasonCode = "UNSUPPORTED_CONFIGURATION"
	// ReasonNotBillable: the resource type never carries a direct AWS charge of
	// its own (its cost, if any, is attributed to other resources or usage this
	// tool does not model), so $0.00 is the correct, final answer.
	ReasonNotBillable ReasonCode = "NOT_BILLABLE"
)

var allReasonCodes = map[ReasonCode]string{
	ReasonUnsupportedType:          "resource type is not in the pricing catalog",
	ReasonMissingAttribute:         "a required attribute is missing from the plan",
	ReasonUnknownAfterApply:        "a required attribute is not known until apply",
	ReasonNoUsageData:              "no usage value was supplied for this component",
	ReasonPricingAPIError:          "the AWS Price List API could not be queried",
	ReasonUnsupportedConfiguration: "this configuration of the resource is not modelled",
	ReasonNotBillable:              "this resource type has no direct AWS cost of its own",
}

// Valid reports whether c is one of the defined reason codes.
func (c ReasonCode) Valid() bool {
	_, ok := allReasonCodes[c]
	return ok
}

// Message returns the default human-readable text for c, or "" if c is unknown.
func (c ReasonCode) Message() string { return allReasonCodes[c] }

// ReasonCodes returns every defined reason code, sorted, for documentation and
// tests.
func ReasonCodes() []ReasonCode {
	out := make([]ReasonCode, 0, len(allReasonCodes))
	for c := range allReasonCodes {
		out = append(out, c)
	}
	sortReasonCodes(out)
	return out
}

func sortReasonCodes(s []ReasonCode) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
