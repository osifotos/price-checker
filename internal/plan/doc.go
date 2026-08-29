// Package plan loads a Terraform plan (a JSON file, stdin, or a directory that
// "terraform show -json" is run against) and turns it into a provider-neutral
// model. Nothing outside this package imports Terraform's JSON shapes: callers
// work with Plan, Resource, and AttrMap.
//
// resource_changes is the source of truth, not planned_values, because only
// resource_changes carries after_unknown — the information about which
// attributes are "known after apply".
package plan
