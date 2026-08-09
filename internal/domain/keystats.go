package domain

type Keystats struct {
	ClosureFinItemsResults  []KeystatsFinGroup
	FinancialYearParent     KeystatsYearParent
	Stats                   KeystatsStats
	Info                    string
	DividendGroup           KeystatsDividendGroup
	FinancialReportCurrency []string
}

type KeystatsFinGroup struct {
	KeystatsName   string
	FinNameResults []KeystatsItem
}

type KeystatsItem struct {
	Fitem          KeystatsFitem
	IsNewUpdate    bool
	HiddenGraphIco bool
}

type KeystatsFitem struct {
	ID    string
	Name  string
	Value string
}

type KeystatsYearParent struct {
	FinancialYearGroups    []KeystatsYearGroup
	FinancialYearGroupsUSD []KeystatsYearGroup
}

type KeystatsYearGroup struct {
	FinancialYearValues []KeystatsYear
}

type KeystatsYear struct {
	Year            string
	PeriodValues    []KeystatsPeriod
	AnnualisedValue string
	TTMValue        string
	IsNewUpdate     bool
	Dividend        string
	PayoutRatio     string
	DividendYield   string
}

type KeystatsPeriod struct {
	Period       string
	Year         string
	QuarterValue string
	IsNewUpdate  bool
}

type KeystatsStats struct {
	CurrentShareOutstanding string
	MarketCap               string
	EnterpriseValue         string
	FreeFloat               string
}

type KeystatsDividendGroup struct {
	FitemID            []string
	DividendYearValues []KeystatsDividendYear
}

type KeystatsDividendYear struct {
	Period      int
	Dividend    string
	ExDate      string
	PaymentDate string
}
