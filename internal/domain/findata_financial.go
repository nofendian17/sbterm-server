package domain

type FindataFinancial struct {
	Currency        []string
	DefaultCurrency string
	RoundingValue   []int
	DataTables      FindataDataTables
}

type FindataDataTables struct {
	Periods      []string
	Accounts     []FindataAccount
	MaxShowLevel int
}

type FindataAccount struct {
	ID                int64
	Level             int
	Name              string
	Values            []string
	Accounts          []FindataAccount
	IsTotalExist      bool
	IsDefaultExpanded bool
	MaxShowLevel      int
}
