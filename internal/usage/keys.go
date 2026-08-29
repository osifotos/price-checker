package usage

// usageKey describes one usage-based component input.
type usageKey struct {
	Key         string
	Description string
}

// keysByType lists the usage-based component inputs each catalogued resource
// type accepts. It drives both skeleton generation and key validation.
var keysByType = map[string][]usageKey{
	"aws_s3_bucket": {
		{"storage_gb", "average GB stored per month (Standard)"},
		{"monthly_tier1_requests", "PUT/COPY/POST/LIST requests per month"},
		{"monthly_tier2_requests", "GET/SELECT and other requests per month"},
	},
	"aws_lambda_function": {
		{"monthly_requests", "invocations per month"},
		{"request_duration_ms", "average execution duration in milliseconds"},
	},
	"aws_nat_gateway": {
		{"monthly_data_processed_gb", "GB processed by the NAT gateway per month"},
	},
	"aws_ebs_snapshot": {
		{"snapshot_size_gb", "size of the snapshot in GB"},
	},
	"aws_ebs_snapshot_copy": {
		{"snapshot_size_gb", "size of the snapshot in GB"},
	},
	"aws_lb": {
		{"monthly_lcu", "average Load Balancer Capacity Units per hour"},
	},
	"aws_alb": {
		{"monthly_lcu", "average Load Balancer Capacity Units per hour"},
	},
	"aws_cloudwatch_log_group": {
		{"monthly_ingested_gb", "GB of logs ingested per month"},
		{"storage_gb", "GB of logs stored per month"},
	},
	"aws_dynamodb_table": {
		{"storage_gb", "GB stored per month"},
		{"monthly_read_request_units", "on-demand read request units per month"},
		{"monthly_write_request_units", "on-demand write request units per month"},
	},
	"aws_rds_cluster": {
		{"storage_gb", "Aurora storage GB per month"},
		{"monthly_io_requests", "Aurora I/O requests per month"},
	},
}

// knownUsageKeys is the flattened set of every accepted usage key.
var knownUsageKeys = func() map[string]bool {
	m := map[string]bool{}
	for _, ks := range keysByType {
		for _, k := range ks {
			m[k.Key] = true
		}
	}
	return m
}()
