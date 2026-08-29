package pricing

import (
	"encoding/json"
	"sort"
)

// productDoc is one entry of a GetProducts PriceList response.
type productDoc struct {
	Product struct {
		SKU        string            `json:"sku"`
		Attributes map[string]string `json:"attributes"`
	} `json:"product"`
	Terms struct {
		OnDemand map[string]struct {
			PriceDimensions map[string]struct {
				Unit         string `json:"unit"`
				Description  string `json:"description"`
				PricePerUnit struct {
					USD string `json:"USD"`
				} `json:"pricePerUnit"`
			} `json:"priceDimensions"`
		} `json:"OnDemand"`
	} `json:"terms"`
}

// parsePriceList converts the raw JSON strings returned by GetProducts into a
// deterministically ordered slice of on-demand PriceDimension values. Reserved
// terms and dimensions with no USD price are skipped.
func parsePriceList(raw []string) ([]PriceDimension, error) {
	var out []PriceDimension
	for _, s := range raw {
		var doc productDoc
		if err := json.Unmarshal([]byte(s), &doc); err != nil {
			return nil, err
		}
		for _, term := range doc.Terms.OnDemand {
			for rateCode, pd := range term.PriceDimensions {
				if pd.PricePerUnit.USD == "" {
					continue
				}
				out = append(out, PriceDimension{
					SKU:         doc.Product.SKU,
					RateCode:    rateCode,
					Unit:        pd.Unit,
					USD:         pd.PricePerUnit.USD,
					Description: pd.Description,
					Attributes:  doc.Product.Attributes,
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SKU != out[j].SKU {
			return out[i].SKU < out[j].SKU
		}
		return out[i].RateCode < out[j].RateCode
	})
	return out, nil
}
