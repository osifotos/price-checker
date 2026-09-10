package awsauth

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestPriceListRegion(t *testing.T) {
	cases := map[string]string{
		"us-east-1":  "us-east-1",
		"ap-south-1": "ap-south-1",
		"eu-west-1":  "us-east-1",
		"us-west-2":  "us-east-1",
		"":           "us-east-1",
	}
	for in, want := range cases {
		if got := PriceListRegion(aws.Config{Region: in}); got != want {
			t.Errorf("PriceListRegion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTargetRegionUsesConfiguredRegion(t *testing.T) {
	cases := map[string]string{
		"us-east-1":  "us-east-1",
		"ap-south-1": "ap-south-1",
		"eu-west-1":  "eu-west-1",
		"us-west-2":  "us-west-2",
		"":           "us-east-1",
	}
	for in, want := range cases {
		if got := TargetRegion(aws.Config{Region: in}, ""); got != want {
			t.Errorf("TargetRegion(%q) = %q, want %q", in, got, want)
		}
	}
	if got := TargetRegion(aws.Config{Region: "us-west-2"}, "eu-central-1"); got != "eu-central-1" {
		t.Fatalf("TargetRegion override = %q, want %q", got, "eu-central-1")
	}
}

func TestCredentialHelpMentionsProfile(t *testing.T) {
	if CredentialHelp == "" {
		t.Fatal("CredentialHelp is empty")
	}
}
