package aws

import (
	"context"
	"math/big"

	"github.com/mtosin123/tf-price_checker/internal/pricing"
	"github.com/mtosin123/tf-price_checker/pkg/schema"
)

// ---- aws_db_instance ----

type rdsInstancePricer struct{}

func (rdsInstancePricer) ResourceTypes() []string { return []string{"aws_db_instance"} }

func (rdsInstancePricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	class, ok := res.Attributes.String("instance_class")
	if !ok {
		out.Skip(res, "Database instance", missingOrUnknown(res.Attributes, "instance_class"))
		return out, nil
	}
	engine := "postgres"
	if e, ok := res.Attributes.String("engine"); ok {
		engine = e
	}
	deployment := "Single-AZ"
	if multiAZ, ok := res.Attributes.Bool("multi_az"); ok && multiAZ {
		deployment = "Multi-AZ"
	}

	switch dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonRDS", RegionCode: in.Region, Purpose: "rds " + class,
		Filters: []pricing.Filter{
			f("instanceType", class),
			f("databaseEngine", rdsEngineName(engine)),
			f("deploymentOption", deployment),
		},
	}); {
	case err != nil:
		out.Skip(res, "Database instance", schema.ReasonPricingAPIError)
	case !found:
		out.Skip(res, "Database instance", schema.ReasonUnsupportedConfiguration)
	default:
		if c, ok := pricing.HourlyComponent("Database instance ("+deployment+", "+class+")", dim.USD); ok {
			out.Add(c)
		}
	}

	// allocated storage
	if gb, ok := res.Attributes.Float("allocated_storage"); ok {
		storageType := "gp2"
		if st, ok := res.Attributes.String("storage_type"); ok {
			storageType = st
		}
		if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: "AmazonRDS", RegionCode: in.Region, Purpose: "rds storage",
			Filters: []pricing.Filter{f("productFamily", "Database Storage"), f("volumeType", rdsVolumeType(storageType)), f("deploymentOption", deployment)},
		}); err == nil && found {
			if c, ok := pricing.Component("Database storage ("+storageType+")", "GB-months", dim.USD, big.NewRat(int64(gb), 1), false); ok {
				out.Add(c)
			}
		}
	}
	return out, nil
}

func rdsEngineName(engine string) string {
	switch engine {
	case "postgres":
		return "PostgreSQL"
	case "mysql":
		return "MySQL"
	case "mariadb":
		return "MariaDB"
	case "oracle-ee", "oracle-se2":
		return "Oracle"
	case "sqlserver-ex", "sqlserver-web", "sqlserver-se", "sqlserver-ee":
		return "SQL Server"
	default:
		return engine
	}
}

func rdsVolumeType(t string) string {
	switch t {
	case "gp2", "gp3":
		return "General Purpose"
	case "io1", "io2":
		return "Provisioned IOPS"
	case "standard":
		return "Magnetic"
	default:
		return "General Purpose"
	}
}

// ---- aws_rds_cluster / aws_rds_cluster_instance (Aurora) ----

type auroraPricer struct{}

func (auroraPricer) ResourceTypes() []string {
	return []string{"aws_rds_cluster", "aws_rds_cluster_instance"}
}

func (auroraPricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	if res.Type == "aws_rds_cluster" {
		// storage and I/O are usage-based for Aurora; the cluster itself has no
		// instance cost.
		out.Skip(res, "Aurora storage", schema.ReasonNoUsageData)
		out.Skip(res, "Aurora I/O", schema.ReasonNoUsageData)
		return out, nil
	}

	class, ok := res.Attributes.String("instance_class")
	if !ok {
		out.Skip(res, "Aurora instance", missingOrUnknown(res.Attributes, "instance_class"))
		return out, nil
	}
	engine := "aurora-postgresql"
	if e, ok := res.Attributes.String("engine"); ok {
		engine = e
	}
	switch dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonRDS", RegionCode: in.Region, Purpose: "aurora " + class,
		Filters: []pricing.Filter{
			f("instanceType", class),
			f("databaseEngine", auroraEngineName(engine)),
			f("deploymentOption", "Single-AZ"),
		},
	}); {
	case err != nil:
		out.Skip(res, "Aurora instance", schema.ReasonPricingAPIError)
	case !found:
		out.Skip(res, "Aurora instance", schema.ReasonUnsupportedConfiguration)
	default:
		if c, ok := pricing.HourlyComponent("Aurora instance ("+class+")", dim.USD); ok {
			out.Add(c)
		}
	}
	return out, nil
}

func auroraEngineName(engine string) string {
	if engine == "aurora-mysql" {
		return "Aurora MySQL"
	}
	return "Aurora PostgreSQL"
}

// ---- aws_elasticache_cluster / aws_elasticache_replication_group ----

type elastiCachePricer struct{}

func (elastiCachePricer) ResourceTypes() []string {
	return []string{"aws_elasticache_cluster", "aws_elasticache_replication_group"}
}

func (elastiCachePricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	nodeType, ok := res.Attributes.String("node_type")
	if !ok {
		out.Skip(res, "Cache nodes", missingOrUnknown(res.Attributes, "node_type"))
		return out, nil
	}
	nodes := int64(1)
	if n, ok := res.Attributes.Float("num_cache_nodes"); ok && n > 0 {
		nodes = int64(n)
	} else if n, ok := res.Attributes.Float("num_node_groups"); ok && n > 0 {
		replicas := int64(0)
		if r, ok := res.Attributes.Float("replicas_per_node_group"); ok {
			replicas = int64(r)
		}
		nodes = int64(n) * (1 + replicas)
	}
	engine := "redis"
	if e, ok := res.Attributes.String("engine"); ok {
		engine = e
	}

	switch dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonElastiCache", RegionCode: in.Region, Purpose: "elasticache " + nodeType,
		Filters: []pricing.Filter{f("instanceType", nodeType), f("cacheEngine", elastiCacheEngineName(engine))},
	}); {
	case err != nil:
		out.Skip(res, "Cache nodes", schema.ReasonPricingAPIError)
	case !found:
		out.Skip(res, "Cache nodes", schema.ReasonUnsupportedConfiguration)
	default:
		if c, ok := pricing.Component("Cache nodes ("+nodeType+" x"+i64(nodes)+")", "node-hours", dim.USD, big.NewRat(schema.MonthlyHours*nodes, 1), false); ok {
			out.Add(c)
		}
	}
	return out, nil
}

func elastiCacheEngineName(e string) string {
	if e == "memcached" {
		return "Memcached"
	}
	return "Redis"
}

// ---- aws_dynamodb_table ----

type dynamoDBPricer struct{}

func (dynamoDBPricer) ResourceTypes() []string { return []string{"aws_dynamodb_table"} }

func (dynamoDBPricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	billing := "PROVISIONED"
	if b, ok := res.Attributes.String("billing_mode"); ok {
		billing = b
	}
	if billing != "PROVISIONED" {
		out.Skip(res, "On-demand read/write requests", schema.ReasonNoUsageData)
		out.Skip(res, "Storage", schema.ReasonNoUsageData)
		return out, nil
	}

	rcu, _ := res.Attributes.Float("read_capacity")
	wcu, _ := res.Attributes.Float("write_capacity")

	if rcu > 0 {
		if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: "AmazonDynamoDB", RegionCode: in.Region, Purpose: "ddb rcu",
			Filters: []pricing.Filter{f("group", "DDB-ReadUnits")},
		}); err == nil && found {
			if c, ok := pricing.Component("Provisioned read capacity", "RCU-hours", dim.USD, big.NewRat(schema.MonthlyHours*int64(rcu), 1), false); ok {
				out.Add(c)
			}
		}
	}
	if wcu > 0 {
		if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: "AmazonDynamoDB", RegionCode: in.Region, Purpose: "ddb wcu",
			Filters: []pricing.Filter{f("group", "DDB-WriteUnits")},
		}); err == nil && found {
			if c, ok := pricing.Component("Provisioned write capacity", "WCU-hours", dim.USD, big.NewRat(schema.MonthlyHours*int64(wcu), 1), false); ok {
				out.Add(c)
			}
		}
	}
	out.Skip(res, "Storage", schema.ReasonNoUsageData)
	return out, nil
}
