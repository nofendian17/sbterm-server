package domain

// SearchResult groups every section the Stockbit symbol search returns.
// Sections are always present (empty when no match), so clients can rely on
// non-nil slices.
type SearchResult struct {
	Chat       []SearchChat
	Company    []SearchCompany
	Insider    []SearchInsider
	People     []SearchPerson
	Sector     []SearchSector
	Industries []SearchIndustry
	Pagination SearchPagination
}

// SearchCompany is one matching company (stock, right or warrant).
type SearchCompany struct {
	ID          string
	Name        string
	Country     string
	Desc        string
	Exchange    string
	IsFollowing bool
	Img         string
	IsVerified  bool
	Other       string
	Status      string
	Symbol2     string
	Symbol3     string
	TotalFollow int
	IsTradeable bool
	Type        string
	URL         string
	IconURL     string
}

// SearchInsider is one matching insider (company or person).
type SearchInsider struct {
	ID          string
	IsFollowing bool
	Label       string
	IsVerified  bool
	Permalink   string
	Tradeable   int
	Type        string
}

// SearchChat is one matching chat room.
type SearchChat struct {
	ID          string
	Label       string
	ChatType    string
	IsFollowing bool
	IsVerified  bool
	IconURL     string
}

// SearchPerson is one matching user.
type SearchPerson struct {
	ID          string
	Label       string
	IsFollowing bool
	IsVerified  bool
	IconURL     string
	Permalink   string
}

// SearchSector is one matching sector.
type SearchSector struct {
	ID       string
	Name     string
	IconURL  string
	IsFollow bool
}

// SearchIndustry is one matching industry.
type SearchIndustry struct {
	ID       string
	Name     string
	IconURL  string
	IsFollow bool
}

// SearchPagination reports whether more matches exist in each section.
type SearchPagination struct {
	HasMoreCompanies bool
	HasMoreInsiders  bool
	HasMoreUsers     bool
}
