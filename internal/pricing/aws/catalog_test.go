package aws

import "testing"

func TestCatalogCoversDocumentedTypes(t *testing.T) {
	cat := NewCatalog()
	want := []string{
		"aws_instance", "aws_ebs_volume", "aws_ebs_snapshot", "aws_ebs_snapshot_copy", "aws_eip",
		"aws_db_instance", "aws_rds_cluster", "aws_rds_cluster_instance",
		"aws_elasticache_cluster", "aws_elasticache_replication_group",
		"aws_dynamodb_table", "aws_lb", "aws_alb", "aws_elb", "aws_nat_gateway",
		"aws_lambda_function", "aws_s3_bucket",
		"aws_eks_cluster", "aws_eks_node_group",
		"aws_cloudwatch_log_group", "aws_cloudwatch_metric_alarm", "aws_cloudwatch_dashboard",
		"aws_sqs_queue",
		"aws_route_table", "aws_security_group", "aws_subnet", "aws_vpc", "aws_ecs_cluster",
	}
	for _, typ := range want {
		if _, ok := cat.For(typ); !ok {
			t.Errorf("catalog missing pricer for %q", typ)
		}
	}
	if _, ok := cat.For("aws_kms_key"); ok {
		t.Error("catalog should not claim aws_kms_key")
	}
	if len(cat.All()) == 0 {
		t.Fatal("catalog is empty")
	}
}

func TestNoDuplicateTypeRegistration(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Pricers() {
		for _, typ := range p.ResourceTypes() {
			if seen[typ] {
				t.Errorf("resource type %q registered by more than one pricer", typ)
			}
			seen[typ] = true
		}
	}
}
