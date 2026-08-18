package domain

// SearchResult groups every section the Stockbit symbol search returns.
// Sections are always present (empty when no match), so clients can rely on
// non-nil slices.
type SearchResult struct {
	Chat       []SearchChat     `json:"chat"`
	Company    []SearchCompany  `json:"company"`
	Insider    []SearchInsider  `json:"insider"`
	People     []SearchPerson   `json:"people"`
	Sector     []SearchSector   `json:"sector"`
	Industries []SearchIndustry `json:"industries"`
	Pagination SearchPagination `json:"pagination"`
}

// SearchCompany is one matching company (stock, right or warrant).
type SearchCompany struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Country     string `json:"country"`
	Desc        string `json:"desc"`
	Exchange    string `json:"exchange"`
	IsFollowing bool   `json:"is_following"`
	Img         string `json:"img"`
	IsVerified  bool   `json:"is_verified"`
	Other       string `json:"other"`
	Status      string `json:"status"`
	Symbol2     string `json:"symbol_2"`
	Symbol3     string `json:"symbol_3"`
	TotalFollow int    `json:"total_followers"`
	IsTradeable bool   `json:"is_tradeable"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	IconURL     string `json:"icon_url"`
}

// SearchInsider is one matching insider (company or person).
type SearchInsider struct {
	ID          string `json:"id"`
	IsFollowing bool   `json:"is_following"`
	Label       string `json:"label"`
	IsVerified  bool   `json:"is_verified"`
	Permalink   string `json:"permalink"`
	Tradeable   int    `json:"tradeable"`
	Type        string `json:"type"`
}

// SearchChat is one matching chat room.
type SearchChat struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	ChatType    string `json:"chat_type"`
	IsFollowing bool   `json:"is_following"`
	IsVerified  bool   `json:"is_verified"`
	IconURL     string `json:"icon_url"`
}

// SearchPerson is one matching user.
type SearchPerson struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	IsFollowing bool   `json:"is_following"`
	IsVerified  bool   `json:"is_verified"`
	IconURL     string `json:"icon_url"`
	Permalink   string `json:"permalink"`
}

// SearchSector is one matching sector.
type SearchSector struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IconURL  string `json:"icon_url"`
	IsFollow bool   `json:"is_follow"`
}

// SearchIndustry is one matching industry.
type SearchIndustry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IconURL  string `json:"icon_url"`
	IsFollow bool   `json:"is_follow"`
}

// SearchPagination reports whether more matches exist in each section.
type SearchPagination struct {
	HasMoreCompanies bool `json:"has_more_companies"`
	HasMoreInsiders  bool `json:"has_more_insiders"`
	HasMoreUsers     bool `json:"has_more_users"`
}
