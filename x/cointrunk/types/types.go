package types

// Equal - function required by protobuf
func (p *PublisherRespectParams) Equal(c *PublisherRespectParams) bool {
	return p.Denom == c.Denom && p.Tax.Equal(c.Tax)
}
