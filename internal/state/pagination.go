package state

// PageParams holds pagination parameters for List* queries.
type PageParams struct {
	Limit  int
	Offset int
}

// DefaultPageParams returns sensible defaults (50 items, no offset).
func DefaultPageParams() PageParams {
	return PageParams{Limit: 50, Offset: 0}
}

// Normalize ensures Limit and Offset are within valid bounds.
func (p PageParams) Normalize() PageParams {
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > 1000 {
		p.Limit = 1000
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}
