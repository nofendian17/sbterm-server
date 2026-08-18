package domain

type MajorHolderData struct {
	IsMore   bool
	Movement []MajorHolderMovement
}

type MajorHolderMovement struct {
	ID             string
	Name           string
	Symbol         string
	Date           string
	Previous       MajorHolderValueChange
	Current        MajorHolderValueChange
	Changes        MajorHolderValueChange
	Marker         string
	IsPosted       bool
	CMHID          string
	Nationality    string
	ActionType     string
	DataSource     MajorHolderDataSource
	PriceFormatted string
	BrokerDetail   MajorHolderBroker
	Badges         []string
}

type MajorHolderValueChange struct {
	Value          string
	Percentage     string
	FormattedValue string
}

type MajorHolderDataSource struct {
	Label string
	Type  string
}

type MajorHolderBroker struct {
	Code  string
	Group string
}
