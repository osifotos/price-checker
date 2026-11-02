# AWS Resource Support Matrix

This document details the AWS Terraform resource support matrix for `price-checker`.

> **Note on AWS Resource Scope**: The official Terraform AWS Provider (`hashicorp/aws`) manages over **1,400+ individual resource types** across 200+ AWS services.
> - **In-Catalog Resources**: Explicitly mapped to pricing algorithms (`SUPPORTED` / `USAGE_FILE_NEEDED`) or recognized as $0.00 infrastructure (`NOT_BILLABLE`).
> - **All Other Resources**: Any `aws_*` resource not explicitly modeled in the catalog is treated automatically as **`NOT_SUPPORTED`** and reported under the **Not Estimated** breakdown with reason `UNSUPPORTED_TYPE`.

---

## Status Tags Legend

| Tag | Meaning | Estimation Behavior |
|---|---|---|
| `SUPPORTED` | Directly supported and priced from Terraform plan attributes. | Priced automatically ($/hr, $/mo). |
| `USAGE_FILE_NEEDED` | Resource cost depends partially or entirely on runtime consumption (requests, data transfer, storage volume). | Fixed baseline priced automatically; usage components require assumptions via `--usage-file`. |
| `NOT_BILLABLE` | Resource has no direct standalone AWS charge (e.g., networking config, IAM metadata). | Evaluated to `$0.00` with reason `NOT_BILLABLE`. |
| `NOT_SUPPORTED` | Resource type is not currently modeled in v1 catalog (all other Terraform `aws_*` types). | Flagged as `UNSUPPORTED_TYPE` in not-estimated report. |

---

## 1. Supported Resources (Compute, Storage, Database, Networking)

| Terraform Resource | Status | Usage File Needed? | Key Usage Keys / Notes |
|---|---|---|---|
| `aws_instance` | `SUPPORTED` | No | Priced from `instance_type`, root block device, public IPv4. |
| `aws_ebs_volume` | `SUPPORTED` | No | Priced from `type` (`gp2`, `gp3`, `io1`, `io2`, `st1`, `sc1`, `standard`), `size`, provisioned `iops`, `throughput`. |
| `aws_ebs_snapshot` | `USAGE_FILE_NEEDED` | Yes (for storage) | `snapshot_size_gb`: GB stored per month. |
| `aws_ebs_snapshot_copy` | `USAGE_FILE_NEEDED` | Yes (for storage) | `snapshot_size_gb`: GB stored per month. |
| `aws_eip` | `SUPPORTED` | No | Standard hourly IPv4 address pricing. |
| `aws_eks_cluster` | `SUPPORTED` | No | Standard EKS control plane cluster hourly fee ($0.10/hr). |
| `aws_eks_node_group` | `SUPPORTED` | No | Priced from `instance_types`, `scaling_config.desired_size`, `disk_size`. |
| `aws_db_instance` | `SUPPORTED` | No | Priced from `instance_class`, `allocated_storage`, `storage_type`, `iops`, `multi_az`. |
| `aws_rds_cluster_instance` | `SUPPORTED` | No | Priced from `instance_class`. |
| `aws_rds_cluster` | `USAGE_FILE_NEEDED` | Optional | Priced from `serverlessv2_scaling_configuration` (min/max ACUs); Usage keys: `storage_gb`, `monthly_io_requests`. |
| `aws_elasticache_cluster` | `SUPPORTED` | No | Priced from `node_type`, `num_cache_nodes`. |
| `aws_elasticache_replication_group` | `SUPPORTED` | No | Priced from `node_type`, `num_cache_clusters`, `replicas_per_node_group`. |
| `aws_dynamodb_table` | `USAGE_FILE_NEEDED` | Optional / Mode dependent | Provisioned mode priced from `read_capacity`, `write_capacity`. On-demand mode usage keys: `storage_gb`, `monthly_read_request_units`, `monthly_write_request_units`. |
| `aws_s3_bucket` | `USAGE_FILE_NEEDED` | Yes | Usage keys: `storage_gb`, `monthly_tier1_requests` (PUT/POST), `monthly_tier2_requests` (GET). |
| `aws_lambda_function` | `USAGE_FILE_NEEDED` | Yes | Usage keys: `monthly_requests` (invocations), `request_duration_ms` (runtime duration). Memory size read from Terraform plan. |
| `aws_nat_gateway` | `USAGE_FILE_NEEDED` | Partially | Fixed hourly charge priced automatically ($0.045/hr). Usage key for data traffic: `monthly_data_processed_gb`. |
| `aws_lb` / `aws_alb` / `aws_elb` | `USAGE_FILE_NEEDED` | Partially | Fixed hourly charge priced automatically (ALB/NLB/Classic). Usage key for capacity: `monthly_lcu`. |
| `aws_cloudwatch_metric_alarm` | `SUPPORTED` | No | Standard metric alarm pricing. |
| `aws_cloudwatch_dashboard` | `SUPPORTED` | No | Dashboard hourly/monthly pricing. |
| `aws_cloudwatch_log_group` | `USAGE_FILE_NEEDED` | Yes | Usage keys: `monthly_ingested_gb`, `storage_gb`. |
| `aws_sqs_queue` | `USAGE_FILE_NEEDED` | Yes | Usage keys: `monthly_requests` (Standard / FIFO). |

---

## 2. Non-Billable Resources ($0.00 Direct AWS Cost)

These resources are natively recognized by `price-checker`. They resolve to `$0.00` and are categorized as `NOT_BILLABLE` (not errors or unsupported types).

| Terraform Resource | Status | Reason / Explanation |
|---|---|---|
| `aws_route_table` | `NOT_BILLABLE` | Virtual routing configuration is free. |
| `aws_route_table_association` | `NOT_BILLABLE` | Subnet routing association is free. |
| `aws_security_group` | `NOT_BILLABLE` | Stateful firewall rule container is free. |
| `aws_security_group_rule` | `NOT_BILLABLE` | Security group rules are free. |
| `aws_subnet` | `NOT_BILLABLE` | Subnet IP allocation within VPC is free. |
| `aws_vpc` | `NOT_BILLABLE` | Virtual Private Cloud infrastructure is free. |
| `aws_internet_gateway` | `NOT_BILLABLE` | Internet Gateway attachments have no base hourly fee. |
| `aws_iam_role` | `NOT_BILLABLE` | Identity & Access Management roles are free. |
| `aws_iam_policy` | `NOT_BILLABLE` | IAM policies are free. |
| `aws_iam_role_policy_attachment` | `NOT_BILLABLE` | IAM attachments are free. |
| `aws_iam_instance_profile` | `NOT_BILLABLE` | IAM instance profiles are free. |
| `aws_ecs_cluster` | `NOT_BILLABLE` | ECS cluster control plane is free (underlying EC2/Fargate tasks are billed). |
| `aws_apigatewayv2_authorizer` | `NOT_BILLABLE` | Authorizer configuration carries no base hourly charge. |
| `aws_apigatewayv2_stage` | `NOT_BILLABLE` | Stage deployment carries no base hourly charge. |

---

## 3. Major Unsupported AWS Resource Families (Marked as `UNSUPPORTED_TYPE`)

The official Terraform AWS provider has **1,400+ resource types**. Below is a breakdown by major AWS service categories that are currently unmodeled (`NOT_SUPPORTED`):

| AWS Service / Domain | Example Terraform Resources | Status | Reason / Future Scope |
|---|---|---|---|
| **Security, Identity & Secrets** | `aws_kms_key`, `aws_kms_replica_key`, `aws_secretsmanager_secret`, `aws_guardduty_detector`, `aws_securityhub_account`, `aws_inspector2_enabler` | `NOT_SUPPORTED` | Key storage fees ($1/mo), secret fees ($0.40/mo), security event ingestion. |
| **API Gateway & App Integration** | `aws_api_gateway_rest_api`, `aws_api_gateway_deployment`, `aws_apigatewayv2_api`, `aws_appsync_graphql_api`, `aws_stepfunctions_state_machine` | `NOT_SUPPORTED` | Request-based billing, state transitions. |
| **Messaging & Notifications** | `aws_sns_topic`, `aws_sns_topic_subscription`, `aws_ses_domain_identity`, `aws_eventbridge_rule` | `NOT_SUPPORTED` | Delivery charges, email sending tiers, event bus evaluations. |
| **DNS, Edge & CDN** | `aws_route53_zone`, `aws_route53_record`, `aws_route53_resolver_endpoint`, `aws_cloudfront_distribution`, `aws_globalaccelerator_accelerator` | `NOT_SUPPORTED` | Hosted zone fee ($0.50/mo), query fees, CDN data egress & edge requests. |
| **Containers & Batch (Managed Compute)** | `aws_ecs_service`, `aws_ecs_task_definition` (Fargate / Fargate Spot), `aws_apprunner_service`, `aws_batch_compute_environment` | `NOT_SUPPORTED` | Fargate vCPU/RAM per second charges, App Runner provisioned memory. |
| **Analytics, Streaming & Search** | `aws_opensearch_domain`, `aws_elasticsearch_domain`, `aws_kinesis_stream`, `aws_kinesis_firehose_delivery_stream`, `aws_msk_cluster`, `aws_emr_cluster`, `aws_glue_job`, `aws_athena_workgroup` | `NOT_SUPPORTED` | Master/data node instances, shard-hours, MSK broker instances, DPU hours. |
| **Data Warehousing & Big Data** | `aws_redshift_cluster`, `aws_redshiftserverless_namespace`, `aws_redshiftserverless_workgroup` | `NOT_SUPPORTED` | Redshift node hourly rates or Serverless RPUs. |
| **Networking & Transit** | `aws_dx_connection`, `aws_vpn_connection`, `aws_ec2_transit_gateway`, `aws_ec2_transit_gateway_vpc_attachment`, `aws_vpc_endpoint` (Interface / PrivateLink), `aws_networkfirewall_firewall` | `NOT_SUPPORTED` | Port hourly charges, attachment fees, PrivateLink endpoint hours + data. |
| **Storage & Backup** | `aws_efs_file_system`, `aws_fsx_lustre_file_system`, `aws_fsx_windows_file_system`, `aws_backup_vault`, `aws_glacier_vault` | `NOT_SUPPORTED` | EFS provisioned throughput/storage, FSx storage/throughput capacity. |
| **Certificates & Web Security** | `aws_acm_certificate`, `aws_acmpca_certificate_authority`, `aws_wafv2_web_acl`, `aws_shield_protection` | `NOT_SUPPORTED` | Private CA monthly fees ($400/mo), WAF WebACL/rule evaluations. |
| **Auth & User Management** | `aws_cognito_user_pool`, `aws_cognito_identity_pool`, `aws_directory_service_directory` | `NOT_SUPPORTED` | Monthly Active Users (MAU), Directory hourly costs. |
| **Developer Tools & CI/CD** | `aws_codebuild_project`, `aws_codepipeline`, `aws_codecommit_repository` | `NOT_SUPPORTED` | Build instance minutes, active pipeline charges. |
| **AI / ML Services** | `aws_sagemaker_endpoint_configuration`, `aws_sagemaker_notebook_instance`, `aws_bedrock_custom_model` | `NOT_SUPPORTED` | ML instance hourly pricing, token consumption. |
| **All Other AWS Resources (1,000+)** | `aws_iot_*`, `aws_quicksight_*`, `aws_config_*`, `aws_connect_*`, `aws_macie2_*`, `aws_transfer_*`, etc. | `NOT_SUPPORTED` | Resource type is recognized as not in catalog and surfaced as `UNSUPPORTED_TYPE`. |

---

## How `price-checker` Handles Unlisted Resources

You do not need to maintain an exhaustive list of every single AWS resource. `price-checker` uses an allowlist-based catalog:

```
                      Terraform Resource
                             │
            ┌────────────────┴────────────────┐
            ▼                                 ▼
   In Pricing Catalog?               Not in Catalog?
      │            │                          │
      ▼            ▼                          ▼
 [SUPPORTED]  [NOT_BILLABLE]          [NOT_SUPPORTED]
   (Priced)     ($0.00 Cost)    (Flagged: UNSUPPORTED_TYPE)
```

1. If the resource is in the catalog, it calculates the cost or marks it `$0.00` (`NOT_BILLABLE`).
2. If the resource is any other `aws_*` resource out of the 1,400+ available, it gracefully handles it by outputting:
   ```text
   Resource: <resource_name>
   Reason:   UNSUPPORTED_TYPE (resource type is not in the pricing catalog)
   ```
3. With the `--strict` flag enabled, the CLI will exit with code `2` if any unsupported resource is present.

---

## AWS Lifecycle / Deprecation Checks

Lifecycle notices are separate from pricing support and do not affect estimate
coverage or `--strict`. The bundled catalog currently flags
`aws_launch_configuration`, which AWS restricts for accounts created on or
after October 1, 2024 and recommends replacing with launch templates. The
catalog can be extended or overridden with `--deprecations-file`; see the
[README](README.md#deprecation-checks) for its YAML format.

When `--path` is a Terraform directory, provider deprecation diagnostics are
also captured from `terraform plan -json`. Plan JSON files and stdin do not
preserve provider diagnostics.


---

## Generating a Usage Template

To generate an initial usage file pre-populated for all usage-dependent resources in your Terraform plan, run:

```bash
price-checker usage generate --path plan.json > infra-usage.yml
```

Then supply it to the estimate run:

```bash
price-checker --path plan.json --usage-file infra-usage.yml
```
