package pricing

import "testing"

const ec2Product = `{
  "product": { "sku": "ABC123", "attributes": { "instanceType": "t3.medium", "regionCode": "us-east-1" } },
  "terms": { "OnDemand": { "ABC123.JRTCKXETXF": { "priceDimensions": {
    "ABC123.JRTCKXETXF.6YS6EN2CT7": { "unit": "Hrs", "description": "$0.0416 per On Demand Linux t3.medium",
      "pricePerUnit": { "USD": "0.0416000000" } } } } } }
}`

const reservedOnlyProduct = `{
  "product": { "sku": "RSV1", "attributes": {} },
  "terms": { "Reserved": { "RSV1.X": { "priceDimensions": {
    "RSV1.X.Y": { "unit": "Hrs", "pricePerUnit": { "USD": "0.02" } } } } } }
}`

const zeroPriceProduct = `{
  "product": { "sku": "FREE1", "attributes": {} },
  "terms": { "OnDemand": { "FREE1.A": { "priceDimensions": {
    "FREE1.A.B": { "unit": "Hrs", "pricePerUnit": { "USD": "" } } } } } }
}`

func TestParsePriceList(t *testing.T) {
	dims, err := parsePriceList([]string{ec2Product, reservedOnlyProduct, zeroPriceProduct})
	if err != nil {
		t.Fatal(err)
	}
	if len(dims) != 1 {
		t.Fatalf("want 1 dimension (reserved and zero-price skipped), got %d", len(dims))
	}
	d := dims[0]
	if d.USD != "0.0416000000" || d.Unit != "Hrs" || d.SKU != "ABC123" {
		t.Fatalf("unexpected dimension: %+v", d)
	}
	if d.Attributes["instanceType"] != "t3.medium" {
		t.Fatalf("attributes not carried: %+v", d.Attributes)
	}
}

func TestParsePriceListDeterministicOrder(t *testing.T) {
	a := `{"product":{"sku":"B"},"terms":{"OnDemand":{"B.1":{"priceDimensions":{"B.1.1":{"unit":"Hrs","pricePerUnit":{"USD":"2"}}}}}}}`
	b := `{"product":{"sku":"A"},"terms":{"OnDemand":{"A.1":{"priceDimensions":{"A.1.1":{"unit":"Hrs","pricePerUnit":{"USD":"1"}}}}}}}`
	dims, err := parsePriceList([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if dims[0].SKU != "A" || dims[1].SKU != "B" {
		t.Fatalf("not sorted by SKU: %v %v", dims[0].SKU, dims[1].SKU)
	}
}
