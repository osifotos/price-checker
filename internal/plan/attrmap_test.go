package plan

import "testing"

func TestAttrMapKnownAfterApply(t *testing.T) {
	p := loadFixture(t, "ec2_s3_natgw.json")
	var web Resource
	for _, r := range p.Resources() {
		if r.Address == "aws_instance.web[0]" {
			web = r
		}
	}

	if it, ok := web.Attributes.String("instance_type"); !ok || it != "t3.medium" {
		t.Errorf("instance_type = %q, %v", it, ok)
	}
	if web.Attributes.IsKnown("arn") {
		t.Error("arn should be known-after-apply (unknown)")
	}
	if _, ok := web.Attributes.String("arn"); ok {
		t.Error("String(arn) should report not-ok for an unknown value")
	}
	if !web.Attributes.IsKnown("instance_type") {
		t.Error("instance_type should be known")
	}
}

func TestAttrMapBlock(t *testing.T) {
	p := loadFixture(t, "ec2_s3_natgw.json")
	var web Resource
	for _, r := range p.Resources() {
		if r.Address == "aws_instance.web[0]" {
			web = r
		}
	}
	rbd, ok := web.Attributes.Block("root_block_device")
	if !ok {
		t.Fatal("expected root_block_device block")
	}
	if vt, ok := rbd.String("volume_type"); !ok || vt != "gp3" {
		t.Errorf("volume_type = %q, %v", vt, ok)
	}
	if sz, ok := rbd.Int("volume_size"); !ok || sz != 30 {
		t.Errorf("volume_size = %d, %v", sz, ok)
	}
	if rbd.IsKnown("volume_id") {
		t.Error("nested volume_id should be unknown")
	}
}

func TestAttrMapNumericAndBool(t *testing.T) {
	p := loadFixture(t, "ec2_s3_natgw.json")
	var db Resource
	for _, r := range p.Resources() {
		if r.Address == "aws_db_instance.main" {
			db = r
		}
	}
	if v, ok := db.Attributes.Int("allocated_storage"); !ok || v != 50 {
		t.Errorf("allocated_storage = %d, %v", v, ok)
	}
	if v, ok := db.Attributes.Bool("multi_az"); !ok || v != true {
		t.Errorf("multi_az = %v, %v", v, ok)
	}
	if v, ok := db.PriorAttributes.Int("allocated_storage"); !ok || v != 20 {
		t.Errorf("prior allocated_storage = %d, %v", v, ok)
	}
}

func TestAttrMapMissingKey(t *testing.T) {
	a := newAttrMap(map[string]any{"x": "y"}, nil)
	if _, ok := a.String("nope"); ok {
		t.Error("missing key should report not-ok")
	}
	if a.Has("nope") {
		t.Error("Has(nope) should be false")
	}
	if !a.IsKnown("nope") {
		t.Error("absent key is trivially known")
	}
}
