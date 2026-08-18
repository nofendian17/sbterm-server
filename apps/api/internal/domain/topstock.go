package domain

type TopStockData struct {
	TopBuy        []TopStockItem
	TopSell       []TopStockItem
	Total         []TopStockItem
	ResponseInfo  TopStockResponseInfo
	DisplayOption TopStockDisplayOption
}

type TopStockItem struct {
	Rank         int
	Code         string
	IconURL      string
	Value        RawFormatted
	Lot          RawFormatted
	Average      RawFormatted
	ForeignValue RawFormatted
	Frequency    RawFormatted
}

type RawFormatted struct {
	Raw       string
	Formatted string
}

type TopStockResponseInfo struct {
	Page           int
	Limit          int
	MaxDayDuration int
	StartDate      string
	EndDate        string
	ValueType      string
}

type TopStockDisplayOption struct {
	BannerMessage      string
	ForeignValueColumn bool
	EnabledValueType   TopStockEnabledValueType
}

type TopStockEnabledValueType struct {
	Gross bool
	Net   bool
	Total bool
}
