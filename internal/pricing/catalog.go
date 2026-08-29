package pricing

// Catalog is the set of v1 pricers and lookup by Terraform resource type.
type Catalog interface {
	For(resourceType string) (Pricer, bool)
	All() []Pricer
}

type catalog struct {
	pricers []Pricer
	byType  map[string]Pricer
}

// NewCatalog builds the catalog from an explicit list of pricers. The list is
// supplied by the aws package via RegisterAll so this package does not import
// internal/pricing/aws (which would be a cycle — aws imports pricing).
func NewCatalog(pricers ...Pricer) Catalog {
	c := &catalog{byType: map[string]Pricer{}}
	for _, p := range pricers {
		c.pricers = append(c.pricers, p)
		for _, t := range p.ResourceTypes() {
			c.byType[t] = p
		}
	}
	return c
}

func (c *catalog) For(resourceType string) (Pricer, bool) {
	p, ok := c.byType[resourceType]
	return p, ok
}

func (c *catalog) All() []Pricer { return c.pricers }
