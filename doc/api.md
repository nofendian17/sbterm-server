# REST API Reference

All endpoints return a single response envelope. Every `/api/v1/*` route proxies the
Stockbit (Exodus) API using the server's own credentials — no client `Authorization`
header is needed.

## Endpoint index

All routes registered in `internal/delivery/http/router.go` are documented below:

| # | endpoint | section |
|---|---|---|
| 1 | `GET /health` | [Health & ops](#get-health) |
| 2 | `GET /api/v1/trending` | [Market data](#get-apiv1trending) |
| 3 | `GET /api/v1/market-mover` | [Market data](#get-apiv1market-mover) |
| 4 | `GET /api/v1/market-session` | [Market data](#get-apiv1market-session) |
| 5 | `GET /api/v1/indexes` | [Market data](#get-apiv1indexes) |
| 6 | `GET /api/v1/sectors` | [Market data](#get-apiv1sectors) |
| 7 | `GET /api/v1/stocks` | [Market data](#get-apiv1stocks) |
| 8 | `GET /api/v1/company/{symbol}/profile` | [Company fundamentals](#get-apiv1companysymbolprofile) |
| 9 | `GET /api/v1/company/{symbol}/subsidiaries` | [Company fundamentals](#get-apiv1companysymbolsubsidiaries) |
| 10 | `GET /api/v1/company/{symbol}/shareholding-composition` | [Company fundamentals](#get-apiv1companysymbolshareholding-composition) |
| 11 | `GET /api/v1/insider/shareholding-network` | [Company fundamentals](#get-apiv1insidershareholding-network) |
| 12 | `GET /api/v1/insider/majorholder` | [Company fundamentals](#get-apiv1insidermajorholder) |
| 13 | `GET /api/v1/market-detector/{symbol}` | [Market detector](#get-apiv1market-detectorsymbol) |
| 14 | `GET /api/v1/top-stock` | [Top stock](#get-apiv1top-stock) |
| 15 | `GET /api/v1/company/{symbol}/corp-actions` | [Company fundamentals](#get-apiv1companysymbolcorp-actions) |
| 16 | `GET /api/v1/company/{symbol}/keystats` | [Company fundamentals](#get-apiv1companysymbolkeystats) |
| 17 | `GET /api/v1/company/{symbol}/price-performance` | [Company fundamentals](#get-apiv1companysymbolprice-performance) |
| 18 | `GET /api/v1/company/{symbol}/chart` | [Company fundamentals](#get-apiv1companysymbolchart) |
| 19 | `GET /api/v1/company/{symbol}/fundachart` | [Company fundamentals](#get-apiv1companysymbolfundachart) |
| 20 | `GET /api/v1/fundachart/metrics` | [Company fundamentals](#get-apiv1fundachartmetrics) |
| 21 | `GET /api/v1/company/{symbol}/financial` | [Company fundamentals](#get-apiv1companysymbolfinancial) |
| 22 | `GET /api/v1/index/{symbol}/summary` | [Index summary](#get-apiv1indexsymbolsummary) |
| 23 | `GET /api/v1/index/{symbol}/chart` | [Index chart (summary + OHLC)](#get-apiv1indexsymbolchart) |
| 24 | `GET /api/v1/company/{symbol}/running-trade-chart` | [Running trade chart](#get-apiv1companysymbolrunning-trade-chart) |
| 25 | `GET /api/v1/company/{symbol}/historical-summary` | [Historical price summary](#get-apiv1companysymbolhistorical-summary) |
| 26 | `GET /api/v1/order-trade/broker/top` | [Top brokers](#get-apiv1order-tradebrokertop) |
| 27 | `GET /api/v1/order-trade/broker/activity-chart` | [Broker activity chart](#get-apiv1order-tradebrokeractivity-chart) |
| 28 | `GET /api/v1/order-trade/broker/activity` | [Broker activity transactions](#get-apiv1order-tradebrokeractivity) |
| 29 | `GET /api/v1/order-trade/broker/activity/historical` | [Broker activity historical](#get-apiv1order-tradebrokeractivityhistorical) |
| 30 | `GET /api/v1/user/{username}/stream` | [Stream](#get-apiv1userusernamestream) |
| 31 | `GET /api/v1/stream/announcement/{stream_id}` | [Stream](#get-apiv1streamannouncementstream_id) |
| 32 | `GET /api/v1/order-trade/running-trade` | [Running trade feed](#get-apiv1order-traderunning-trade) |
| 33 | `GET /api/v1/company/{symbol}/orderbook` | [Order book](#get-apiv1companysymbolorderbook) |
| 34 | `GET /api/v1/order-trade/foreign-domestic/historical` | [Foreign-domestic historical](#get-apiv1order-tradeforeign-domestichistorical) |
| 35 | `GET /api/v1/search` | [Search](#get-apiv1search) |
| 36 | `GET /api/v1/order-trade/order-queue` | [Order queue](#get-apiv1order-tradeorder-queue) |
| 37 | `GET /api/v1/notifications` | [Notifications](#get-apiv1notifications) |
| 38 | `GET /api/v1/stream/conversation/{stream_id}` | [Stream](#get-apiv1streamconversationstream_id) |

All routes registered in `internal/delivery/http/router.go` are covered by the
sections below.

## Response envelope

```json
{
  "success": true,
  "message": "optional message",
  "data": { },
  "meta": { "page": 1, "limit": 20, "total_items": 100, "total_pages": 5 },
  "error": { "code": "INTERNAL_ERROR", "message": "...", "details": {} }
}
```

Error codes: `BAD_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`,
`VALIDATION_ERROR`, `INTERNAL_ERROR`, `TOO_MANY_REQUESTS`.

> **Note on examples:** response examples are captured from live calls but are
> abbreviated — arrays and long strings are trimmed with `...`. Treat them as
> illustrative shapes, not exact values.

- `200` success → `success: true`
- `422` invalid input → `error.code: VALIDATION_ERROR` with per-field `details`
- `429` rate limit → `TOO_MANY_REQUESTS` + `Retry-After` header
- `500` upstream/app failure → `INTERNAL_ERROR`

## Health & ops

### `GET /health`
Liveness + DB/Redis connectivity. Returns `200` when both PostgreSQL and Redis
are reachable, `503` when either is down (`status: degraded`).

`data: { status, database, redis }` where `database`/`redis` are `up` or `down`.

#### Example: request / response

```bash
curl 'http://localhost:8080/health'
```

```json
{
  "success": true,
  "data": { "status": "ok", "database": "up", "redis": "up" }
}
```

## Market data

### `GET /api/v1/trending`
Top trending stocks. `data: [{ symbol, name, last, change, percent, previous, logo, status }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/trending'
```

```json
{
  "success": true,
  "data": [
    {
      "symbol": "DSSA",
      "name": "Dian Swastatika Sentosa Tbk",
      "last": "985",
      "change": "+10",
      "percent": "1.03000",
      "previous": "975",
      "logo": "https://assets.stockbit.com/logos/companies/DSSA.png",
      "status": "STATUS_ACTIVE"
    }
  ]
}
```

### `GET /api/v1/market-mover`
Market movers by type.

| param | required | values |
|---|---|---|
| `mover_type` | yes | `MOVER_TYPE_TOP_GAINER`, `MOVER_TYPE_TOP_LOSER`, `MOVER_TYPE_TOP_VALUE`, `MOVER_TYPE_TOP_VOLUME`, `MOVER_TYPE_TOP_FREQUENCY`, `MOVER_TYPE_NET_FOREIGN_BUY`, `MOVER_TYPE_NET_FOREIGN_SELL`, `MOVER_TYPE_IEVAL_TOP_GAINER` |
| `filter_stocks` | no | repeatable; `FILTER_STOCKS_TYPE_{MAIN,DEVELOPMENT,ACCELERATION,NEW_ECONOMY,SPECIAL_MONITORING}_BOARD`, `FILTER_STOCKS_TYPE_WARRANT_AND_RIGHT` |

`data: [{ symbol, name, price, change_value, change_percent, value, volume, freq, net_foreign_buy, net_foreign_sell, iep, iev, ieval, iep_change_prev }]`

#### Example: board filters

```bash
# Top gainers on the main board
curl 'http://localhost:8080/api/v1/market-mover?mover_type=MOVER_TYPE_TOP_GAINER&filter_stocks=FILTER_STOCKS_TYPE_MAIN_BOARD'

# Net foreign buy across all boards (filter optional, repeatable)
curl 'http://localhost:8080/api/v1/market-mover?mover_type=MOVER_TYPE_NET_FOREIGN_BUY'
curl 'http://localhost:8080/api/v1/market-mover?mover_type=MOVER_TYPE_TOP_VALUE&filter_stocks=FILTER_STOCKS_TYPE_MAIN_BOARD&filter_stocks=FILTER_STOCKS_TYPE_DEVELOPMENT_BOARD'
```

- `filter_stocks` can be repeated (`?filter_stocks=A&filter_stocks=B`) to combine boards.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/market-mover?mover_type=MOVER_TYPE_TOP_GAINER'
```

```json
{
  "success": true,
  "data": [
    {
      "symbol": "BBRIBQCX6A",
      "name": "Call Waran BBRI BQ",
      "price": 34,
      "change_value": 32,
      "change_percent": 1600,
      "value": 1786900,
      "volume": 510,
      "freq": 19,
      "net_foreign_buy": 0,
      "net_foreign_sell": 0,
      "iep": 0,
      "iev": 0,
      "ieval": 0,
      "iep_change_prev": 0
    }
  ]
}
```

### `GET /api/v1/market-session`
Current / upcoming market session. `data: { datetime, fca, regular }` — `fca` and
`regular` are session states with `{ state_name, is_last_session, is_end_of_day,
state_start_time, state_end_time, time_left }`.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/market-session'
```

```json
{
  "success": true,
  "data": {
    "datetime": "2026-08-10 19:15:20",
    "fca": {
      "state_name": "STATE_NAME_MARKET_CLOSED",
      "is_last_session": true,
      "is_end_of_day": true,
      "state_start_time": "16:30:00",
      "state_end_time": "23:59:00",
      "time_left": "13 jam 29 menit 40 detik"
    },
    "regular": {
      "state_name": "STATE_NAME_MARKET_CLOSED",
      "is_last_session": true,
      "is_end_of_day": true,
      "state_start_time": "16:30:00",
      "state_end_time": "23:59:59",
      "time_left": "13 jam 29 menit 40 detik"
    }
  }
}
```

`datetime` and `time_left` are live snapshots and change on every call.

### `GET /api/v1/indexes`
IDX index list. `data: { main: [{symbol,name,last,change,percent,marketcap}], all: [...] }`

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/indexes'
```

```json
{
  "success": true,
  "data": {
    "main": [
      { "symbol": "IDX30", "name": "IDX30", "last": "355.760009765625", "change": "-3.64", "percent": "-1.01", "marketcap": "NA" }
    ],
    "all": [
      { "symbol": "ABX", "name": "Papan Akselerasi", "last": "2755.25", "change": "17.73", "percent": "0.65", "marketcap": "NA" }
    ]
  }
}
```

### `GET /api/v1/sectors`
Sector indexes with nested constituent companies.

`data: [{ symbol, icon, type, last, change, percent, companies: [{ symbol, name, last, change, percent, volume, value, marketcap, icon_url, company_status, is_uma }] }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/sectors'
```

```json
{
  "success": true,
  "data": [
    {
      "symbol": "IDXBASIC",
      "icon": "https://assets.stockbit.com/images/IDXBASIC.png",
      "type": "Index",
      "last": 1767.615,
      "change": "-3.50",
      "percent": -0.2,
      "companies": [
        {
          "symbol": "ANTM",
          "name": "Aneka Tambang Tbk.",
          "last": "3140",
          "change": "-20.00",
          "percent": "-0.63",
          "volume": 107393800,
          "value": 1692273616,
          "marketcap": "75456601236500.00"
        }
      ]
    }
  ]
}
```

### `GET /api/v1/stocks`
IHSG constituent list. `data: [{ symbol, name, last, change, percent, volume, value, marketcap, icon_url, company_status, is_uma }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/stocks'
```

```json
{
  "success": true,
  "data": [
    {
      "symbol": "BBCA",
      "name": "Bank Central Asia Tbk.",
      "last": "6375",
      "change": "0.00",
      "percent": "0.00",
      "volume": 88666100,
      "value": -101290572,
      "marketcap": "785878443750000.00",
      "icon_url": "https://assets.stockbit.com/logos/companies/BBCA.png",
      "company_status": "STATUS_ACTIVE",
      "is_uma": "False"
    }
  ]
}
```

## Company fundamentals

### `GET /api/v1/company/{symbol}/profile`
Company profile. `symbol` is path param (required, uppercase as traded).

`data: { background, history, key_executive, address, subsidiary, beneficiary, shareholder, shareholder_director_commissioner, shareholder_numbers, shareholder_one_percent }`

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/company/BBCA/profile'
```

```json
{
  "success": true,
  "data": {
    "background": "PT Bank Central Asia Tbk. atau BBCA ...",
    "history": { "amount": "927 B", "board": "Papan Utama", "date": "31 May 2000", "price": "1,400", "shares": "662,400,000", "free_float": "42.46%" },
    "key_executive": {
      "commissioner": [{ "id": "21223145", "key": "Commissioner", "value": "TONNY KUSNADI" }],
      "director": [{ "id": "21223159", "key": "Director", "value": "DAVID FORMULA" }],
      "independent_commissioner": [{ "id": "21223147", "key": "Commissioner (Independent)", "value": "SUMANTRI SLAMET" }]
    },
    "address": [{ "office": "Menara BCA, Grand Indonesia ...", "phone": "021-23588000", "fax": "021-23588300", "email": ["investor_relations@bca.co.id"], "website": "www.bca.co.id" }],
    "subsidiary": [{ "company": "PT Asuransi Jiwa BCA", "percentage": "90.00%", "types": "Asuransi Jiwa", "value": "1736177" }],
    "beneficiary": [{ "name": "ROBERT BUDI HARTONO" }],
    "shareholder": [{ "percentage": "54.942%", "name": "PT DWIMURIA INVESTAMA ANDALAN", "value": "67.73 B", "badges": ["pengendali"] }],
    "shareholder_director_commissioner": [{ "percentage": "0.03%", "name": "JAHJA SETIAATMADJA", "value": "35.80 M", "badges": ["komisaris"] }]
  }
}
```

### `GET /api/v1/company/{symbol}/subsidiaries`
Subsidiary list.

`data: { currency, last_updated_period, unit, subsidiaries: [{ company_name, business_type, location, commercial_year, total_assets, percentage, operational_status, period, raw }] }`

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/company/BBCA/subsidiaries'
```

```json
{
  "success": true,
  "data": {
    "currency": "CURRENCY_IDR",
    "last_updated_period": "Q2 2026",
    "unit": "UNIT_MILLION",
    "subsidiaries": [
      {
        "company_name": "PT Bank Digital BCA",
        "business_type": "Perbankan",
        "location": "Jakarta",
        "commercial_year": "1965",
        "total_assets": "20,818,005",
        "percentage": "100.00",
        "operational_status": "",
        "period": ""
      }
    ]
  }
}
```

### `GET /api/v1/company/{symbol}/shareholding-composition`
Insider shareholding composition per reporting period.

| param | required | notes |
|---|---|---|
| `symbol` | path | |
| `period_start` | no | `YYYY-MM-DD`, filters upstream periods |
| `period_end` | no | `YYYY-MM-DD` |

`data: [{ report_date, total_shares: {raw, formatted}, compositions: [{ label, shares, percentage, colors }] }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/company/BBCA/shareholding-composition'
```

```json
{
  "success": true,
  "data": [
    {
      "report_date": "2026-07-31",
      "total_shares": { "raw": "123275050000", "formatted": "123.28B" },
      "compositions": [
        {
          "label": "DWIMURIA INVESTAMA ANDALAN",
          "shares": { "raw": "67729950000", "formatted": "67.73B" },
          "percentage": { "raw": 54.94213954891927, "formatted": "54.94%" },
          "colors": { "light": "#0BA16B", "dark": "#0BA16B" }
        }
      ]
    }
  ]
}
```

### `GET /api/v1/insider/shareholding-network`
Shareholding network graph for a root node.

| param | required | values |
|---|---|---|
| `root_id` | yes | node id |
| `root_type` | yes | `SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR` (or `_COMPANY`) |
| `max_depth` | no | int (default upstream behavior) |
| `max_edge_per_node` | no | int |

`data: { root_id, root_type, report_date, nodes: [{id, node_type, metadata: {company|investor}, min_depth, is_rendered}], edges: [{from_id, to_id, shareholding, is_rendered}] }`

#### Example: graph bounds

```bash
# root_id is required; set root_type to a supported node type (see table above)
curl 'http://localhost:8080/api/v1/insider/shareholding-network?root_id=12345&root_type=SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR'

# Cap graph depth and branches per node (optional)
curl 'http://localhost:8080/api/v1/insider/shareholding-network?root_id=12345&root_type=SHAREHOLDING_NETWORK_NODE_TYPE_COMPANY&max_depth=3&max_edge_per_node=10'
```

- Empty `root_id` → `422 VALIDATION_ERROR` (`{"root_id":"is required"}`). `root_type`
  is only validated as required server-side; pass a supported node type or the
  upstream call may fail.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/insider/shareholding-network?root_id=54&root_type=SHAREHOLDING_NETWORK_NODE_TYPE_COMPANY'
```

```json
{
  "success": true,
  "data": {
    "root_id": "company:54",
    "root_type": "SHAREHOLDING_NETWORK_NODE_TYPE_COMPANY",
    "report_date": "31 Jul 26",
    "nodes": [
      {
        "id": "company:54",
        "node_type": "SHAREHOLDING_NETWORK_NODE_TYPE_COMPANY",
        "metadata": {
          "company": { "id": 54, "symbol": "BBCA", "name": "Bank Central Asia Tbk.", "icon_url": "https://assets.stockbit.com/logos/companies/BBCA.png" },
          "investor": null
        },
        "min_depth": 0,
        "is_rendered": true
      }
    ],
    "edges": []
  }
}
```

### `GET /api/v1/insider/majorholder`
Major-holder movement log (paginated).

| param | required | values |
|---|---|---|
| `symbols` | yes | one or more symbols |
| `action_type` | no | `ACTION_TYPE_{UNSPECIFIED,BUY,SELL,CROSS,TRANSFER,CORPACTION}` |
| `source_type` | no | `SOURCE_TYPE_{UNSPECIFIED,KSEI,IDX}` |
| `page` | no | default 1 |
| `limit` | no | default upstream (20) |

`data: { is_more, movement: [{ id, name, symbol, date, previous, current, changes, marker, is_posted, cmh_id, nationality, action_type, data_source, price_formatted, broker_detail, badges }] }`

#### Example: filters and paging

```bash
# Movement log for one or more symbols (comma-separated)
curl 'http://localhost:8080/api/v1/insider/majorholder?symbols=BBCA,BBRI&page=1&limit=10'

# Filter by action type + data source + paging
curl 'http://localhost:8080/api/v1/insider/majorholder?symbols=BBCA&action_type=ACTION_TYPE_BUY&source_type=SOURCE_TYPE_KSEI&page=2&limit=20'
```

- `page`/`limit` default to upstream values (limit 20); `is_more` indicates whether
  more pages exist.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/insider/majorholder?symbols=BBCA&page=1&limit=2'
```

```json
{
  "success": true,
  "data": {
    "is_more": true,
    "movement": [
      {
        "id": "1000000439",
        "name": "HENDRA TANUMIHARDJA",
        "symbol": "BBCA",
        "date": "25 Mar 26",
        "previous": { "value": "193,206", "percentage": "0.0002", "formatted_value": "" },
        "current": { "value": "341,139", "percentage": "0.0003", "formatted_value": "" },
        "changes": { "value": "+147,933", "percentage": "+0.0001", "formatted_value": "147,933" },
        "marker": ""
      }
    ]
  }
}
```

### `GET /api/v1/market-detector/{symbol}`
Bandar detector aggregates and broker buy/sell summaries for a symbol over a date
range. Proxies `/marketdetectors/{symbol}`.

| param | required | values |
|---|---|---|
| `symbol` | path | |
| `from` | yes | `YYYY-MM-DD` — must be earlier than `to` |
| `to` | yes | `YYYY-MM-DD` |
| `transaction_type` | no | `TRANSACTION_TYPE_GROSS`, `TRANSACTION_TYPE_NET` (default) |
| `market_board` | no | `MARKET_BOARD_ALL`, `MARKET_BOARD_REGULER` (default), `MARKET_BOARD_TUNAI`, `MARKET_BOARD_NEGO` |
| `investor_type` | no | `INVESTOR_TYPE_ALL` (default), `INVESTOR_TYPE_DOMESTIC`, `INVESTOR_TYPE_FOREIGN` |
| `limit` | no | int — caps `brokers_buy`/`brokers_sell` list length |

`from` must be earlier than `to` (upstream rejects the inverse). `data.bandar_detector`
holds the accumulation/distribution aggregates (`avg`, `avg5`, `top1`, `top3`, `top5`,
`top10` each with `accdist`/`amount`/`percent`/`vol`), plus `broker_accdist`,
`number_broker_buysell`, `total_buyer`, `total_seller`, `value`, `volume`, `average`.
`data.broker_summary` holds `brokers_buy`/`brokers_sell` (all numeric fields are strings)
and echoes `symbol`/`from`/`to`.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10&limit=25'
```

```json
{
  "success": true,
  "data": {
    "symbol": "BRPT",
    "from": "2026-08-03",
    "to": "2026-08-10",
    "bandar_detector": {
      "average": 1917.1906,
      "avg": { "accdist": "Neutral", "amount": -628582850, "percent": -0.60751265, "vol": -3278.6667 },
      "avg5": { "accdist": "Neutral", "amount": 2635370000, "percent": 2.5470319, "vol": 13746 },
      "broker_accdist": "Dist",
      "number_broker_buysell": 12,
      "top1": { "accdist": "Normal Acc", "amount": 13157295000, "percent": 12.71626, "vol": 68628 },
      "top3": { "accdist": "Neutral", "amount": 1623285200, "percent": 1.5688723, "vol": 8467 },
      "top5": { "accdist": "Neutral", "amount": -6144404000, "percent": -5.938442, "vol": -32049 },
      "top10": { "accdist": "Neutral", "amount": -5922968600, "percent": -5.724429, "vol": -30894 },
      "total_buyer": 40,
      "total_seller": 28,
      "value": 103468280000,
      "volume": 539687
    },
    "broker_summary": {
      "brokers_buy": [
        { "blot": "166709.99", "blotv": "1.67e+07", "bval": "3.18e+10", "bvalv": "3.20e+10", "netbs_broker_code": "BB", "netbs_buy_avg_price": "1910.60", "netbs_date": "20260810", "netbs_stock_code": "BRPT", "type": "Lokal", "freq": "1562" }
      ],
      "brokers_sell": [
        { "netbs_broker_code": "ZP", "netbs_date": "20260810", "netbs_sell_avg_price": "1924.95", "netbs_stock_code": "BRPT", "slot": "-98082", "slotv": "1.17e+07", "sval": "-1.89e+10", "svalv": "2.26e+10", "type": "Asing", "freq": "2374" }
      ]
    }
  }
}
```

### `GET /api/v1/top-stock`
Top buy/sell leaderboards over a date range (proxies `/order-trade/top-stock`).

| param | required | values |
|---|---|---|
| `start` | yes | `YYYY-MM-DD` — must be earlier than `end` |
| `end` | yes | `YYYY-MM-DD` |
| `investor_type` | no | `INVESTOR_TYPE_ALL` (default), `INVESTOR_TYPE_FOREIGN`, `INVESTOR_TYPE_DOMESTIC` |
| `market_type` | no | `MARKET_TYPE_ALL` (default), `MARKET_TYPE_REGULER`, `MARKET_TYPE_TUNAI`, `MARKET_TYPE_NEGO` |
| `value_type` | no | `VALUE_TYPE_NET` (default), `VALUE_TYPE_GROSS`, `VALUE_TYPE_TOTAL` |
| `page` | no | int (default 1) |

`data.top_buy` / `data.top_sell` are ranked lists of `{rank, code, icon_url, value, lot, average, foreign_value, frequency}` where each numeric field is `{raw, formatted}`. `data.response_info` echoes paging/display metadata (`page`, `limit`, `max_day_duration`, `start_date`, `end_date`, `value_type`). `data.display_option` signals which value columns are enabled (`enabled_value_type: {gross, net, total}`).

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/top-stock?start=2026-08-09&end=2026-08-10&limit=25'
```

```json
{
  "success": true,
  "data": {
    "top_buy": [
      {
        "rank": 1,
        "code": "DSSA",
        "icon_url": "https://assets.stockbit.com/logos/companies/DSSA.png",
        "value": { "raw": "1297165000000", "formatted": "1,297.2B" },
        "lot": { "raw": "13305000", "formatted": "13.3M" },
        "average": { "raw": "974", "formatted": "974" },
        "foreign_value": { "raw": "0", "formatted": "0" },
        "frequency": { "raw": "3", "formatted": "3" }
      }
    ],
    "top_sell": [
      {
        "rank": 1,
        "code": "AMRT",
        "icon_url": "https://assets.stockbit.com/logos/companies/AMRT.png",
        "value": { "raw": "-28256436000", "formatted": "-28.3B" },
        "lot": { "raw": "-204165", "formatted": "-204.2K" },
        "average": { "raw": "1384", "formatted": "1,384" },
        "foreign_value": { "raw": "0", "formatted": "0" },
        "frequency": { "raw": "-4", "formatted": "-4" }
      }
    ],
    "response_info": { "page": 1, "limit": 100, "max_day_duration": 360, "start_date": "2026-08-09", "end_date": "2026-08-10", "value_type": "VALUE_TYPE_NET" },
    "display_option": { "banner_message": "", "foreign_value_column": false, "enabled_value_type": { "gross": true, "net": true, "total": true } }
  }
}
```

### `GET /api/v1/company/{symbol}/corp-actions`
Corporate action history. `action_info` is a typed payload dispatched by `action_type`.

| param | required | notes |
|---|---|---|
| `symbol` | path | |
| `limit` | no | int |

`data: [{ action_type, action_info: { rups | rightissue | stocksplit | <other>: {...} } }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/company/BBCA/corp-actions'
```

```json
{
  "success": true,
  "data": [
    {
      "action_type": "dividend",
      "action_info": {
        "dividend": {
          "company_id": "54",
          "company_symbol": "BBCA",
          "corp_action_active": false,
          "dividend_created": "2026-06-05",
          "dividend_cumdate": "2026-06-15",
          "dividend_exdate": "2026-06-17",
          "dividend_id": "117860"
        }
      }
    }
  ]
}
```

### `GET /api/v1/company/{symbol}/keystats`
Key-stat ratios (display-formatted strings, ~10 years).

| param | required | notes |
|---|---|---|
| `symbol` | path | |
| `year_limit` | no | int |

`data: { closure_fin_items_results, financial_year_parent, stats, info, dividend_group, financial_report_currency }`

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/company/BBCA/keystats'
```

```json
{
  "success": true,
  "data": {
    "closure_fin_items_results": [
      {
        "keystats_name": "Valuation",
        "fin_name_results": [
          { "fitem": { "id": "12148", "name": "Current PE Ratio (Annualised)", "value": "13.30" }, "is_new_update": false, "hidden_graph_ico": false }
        ]
      }
    ],
    "financial_year_parent": {
      "financial_year_groups": [
        {
          "financial_year_values": [
            { "year": "2026", "period_values": [{ "period": "Q1", "year": "2026", "quarter_value": "14,684 B", "is_new_update": false }], "annualised_value": "59,069 B", "ttm_value": "58,055 B", "dividend": "301.00", "payout_ratio": "62.82%", "dividend_yield": "4.72%" }
          ]
        }
      ]
    },
    "stats": { "current_share_outstanding": "123.28 B", "market_cap": "785,878 B", "enterprise_value": "766,567 B", "free_float": "42.46%" },
    "info": "",
    "dividend_group": { "fitem_id": ["21507"], "dividend_year_values": [{ "period": 2026, "dividend": "20.00", "ex_date": "17 Jun 26", "payment_date": "26 Jun 26" }] },
    "financial_report_currency": ["IDR"]
  }
}
```

### `GET /api/v1/company/{symbol}/price-performance`
Price performance per timeframe. `data: { prices: [{ close, high, low, percentage, timeframe }] }`
(timeframes: `1D 1W 1M 3M 6M YTD 1Y`).

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/company/BBCA/price-performance'
```

```json
{
  "success": true,
  "data": {
    "prices": [
      {
        "close": { "raw": 6375, "formatted": "6,375" },
        "high": { "raw": 6450, "formatted": "6,450" },
        "low": { "raw": 6350, "formatted": "6,350" },
        "percentage": { "raw": 0, "formatted": "(0.00%)" },
        "timeframe": "1D"
      }
    ]
  }
}
```

### `GET /api/v1/company/{symbol}/chart`
OHLC price bars for building candlestick/line charts (proxies Stockbit Chartbit).
`timeframe` selects the upstream path; `from`/`to` format depends on it:

| param | required | values |
|---|---|---|
| `symbol` | path | |
| `timeframe` | yes | `daily` or `intraday` |
| `from` | yes | `daily`: `YYYY-MM-DD` · `intraday`: Unix seconds |
| `to` | yes | same format as `from` |
| `limit` | intraday only | int ≥ 1 |

- `from` and `to` are **required for both timeframes** (422 otherwise); the
  upstream API returns an empty `chartbit` or an error without them.
- **`daily`** covers 1D/1W/1M aggregates. The upstream pages **backward**: `from`
  must be the newer date and `to` the older one, otherwise the API returns an
  empty `chartbit`. Deep history requires chaining on `last_data`/`previous_timestamp`
  from the response (not yet aggregated by this server).
- **`intraday`** covers minute/hour intervals (1m–4H). `limit` is **required and
  must be ≥ 1** (422 otherwise); without it the upstream API returns an empty
  `chartbit`. Add `minutes_multiplier` for hourly aggregation when needed.

`data: { allow_decimal, chartbit: [{ date, unixdate, datetime, unix_timestamp, open, high, low, close, volume, value, frequency, foreignbuy, foreignsell, foreignflow, soxclose, dividend, shareoutstanding, freq_analyzer, lot, foreign_buy, foreign_sell, symbol }] }`

`daily` bars populate `date`/`unixdate` and the foreign-flow fields; `intraday`
bars populate `datetime`/`unix_timestamp`/`symbol` instead. Missing fields are `0`
or `""`.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/company/BBCA/chart?timeframe=daily&from=2026-08-07&to=2026-07-07'
```

```json
{
  "success": true,
  "data": {
    "allow_decimal": 0,
    "chartbit": [
      {
        "date": "2026-08-07",
        "unixdate": 1786035600,
        "datetime": "",
        "unix_timestamp": "",
        "open": 6375,
        "high": 6450,
        "low": 6325,
        "close": 6375,
        "volume": 111930800,
        "value": 714507295000,
        "frequency": 19833,
        "foreignbuy": 448415995000,
        "foreignsell": 541491165000,
        "foreignflow": -55371844504510,
        "soxclose": 785878443750000,
        "dividend": 0,
        "shareoutstanding": 123275050000,
        "freq_analyzer": 14.347768876433532,
        "lot": 0,
        "foreign_buy": 0,
        "foreign_sell": 0,
        "symbol": ""
      }
    ]
  }
}
```

#### Example: common chart ranges (daily)

Because the upstream pages **backward**, `from` is always the newer date and `to`
the older one — i.e. `from` = today (or the latest trading day) and `to` = today
minus the range. Example calls (dates as of 2026-08-07):

| range | `from` | `to` | curl |
|---|---|---|---|
| 1M | `2026-08-07` | `2026-07-07` | `curl 'http://localhost:8080/api/v1/company/BBCA/chart?timeframe=daily&from=2026-08-07&to=2026-07-07'` |
| 3M | `2026-08-07` | `2026-05-07` | `curl 'http://localhost:8080/api/v1/company/BBCA/chart?timeframe=daily&from=2026-08-07&to=2026-05-07'` |
| 6M | `2026-08-07` | `2026-02-07` | `curl 'http://localhost:8080/api/v1/company/BBCA/chart?timeframe=daily&from=2026-08-07&to=2026-02-07'` |
| 1Y | `2026-08-07` | `2025-08-07` | `curl 'http://localhost:8080/api/v1/company/BBCA/chart?timeframe=daily&from=2026-08-07&to=2025-08-07'` |

To generate the `to` date dynamically from a client:

```bash
# GNU date (Linux)
TO=$(date -d '3 months ago' +%F)
FROM=$(date +%F)
curl "http://localhost:8080/api/v1/company/BBCA/chart?timeframe=daily&from=$FROM&to=$TO"

# BSD date (macOS)
TO=$(date -v-3m +%F)
FROM=$(date +%F)
curl "http://localhost:8080/api/v1/company/BBCA/chart?timeframe=daily&from=$FROM&to=$TO"
```

> If `date +%F` lands on a weekend/holiday (non-trading day), use the most
> recent trading day as `from` instead — e.g. fall back to the previous Friday
> — since a non-trading-day `from` is untested upstream behavior and may return
> an empty or partial `chartbit`.

### `GET /api/v1/company/{symbol}/fundachart`
Raw historical ratio series for one or more fin-items. **This is the raw-number source
for keystats items.**

| param | required | values |
|---|---|---|
| `symbol` | path | |
| `item` | yes | fin-item id(s), comma-separated e.g. `2661,2525,1562` |
| `timeframe` | no | `1y 3y 5y 10y` (default `10y`) |

`data: [{ company_id, company_name, ratios: [{ decimal_point, group_data, item_id, item_name, item_type, suffix, xaxis_id, yaxis_id, chart_data: [{ date, formated_date, value, ratio_value }] }] }]`

#### Example: multiple items

```bash
# Fetch several items at once (comma-separated) over a 5-year window
curl 'http://localhost:8080/api/v1/company/BBCA/fundachart?item=2661,2525&timeframe=5y'

# List available item_id values
curl 'http://localhost:8080/api/v1/fundachart/metrics?metric_name=fundachart'
```

- `timeframe` optional (`1y 3y 5y 10y`, default `10y`); `item` required.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/company/BBCA/fundachart?item=2661&timeframe=5y'
```

```json
{
  "success": true,
  "data": [
    {
      "company_id": 54,
      "company_name": "BBCA",
      "ratios": [
        {
          "decimal_point": 2,
          "group_data": false,
          "item_id": 2661,
          "item_name": "Price",
          "item_type": 6,
          "suffix": "",
          "xaxis_id": 7,
          "yaxis_id": 5,
          "chart_data": [
            { "date": 1632934800, "formated_date": "2021-09-30", "value": 26093371000000, "ratio_value": 26093371000000 }
          ]
        }
      ]
    }
  ]
}
```

### `GET /api/v1/fundachart/metrics`
Catalog of available `item` ids (recursive 3-level tree) to use with `/fundachart`.

| param | required | values |
|---|---|---|
| `metric_name` | yes | `fundachart` |

`data: [{ fitem_id, fitem_name, show_chart_icon, child: [...] }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/fundachart/metrics?metric_name=fundachart'
```

```json
{
  "success": true,
  "data": [
    {
      "fitem_id": 18,
      "fitem_name": "Size",
      "show_chart_icon": 0,
      "child": [
        { "fitem_id": 2892, "fitem_name": "Market Cap", "show_chart_icon": 0, "child": [] }
      ]
    }
  ]
}
```

### `GET /api/v1/company/{symbol}/financial`
Structured financial statement (II-a). No HTML; `data_tables` is the parsed table.

| param | required | values |
|---|---|---|
| `symbol` | path | |
| `data_type` | no | int (upstream: `1` returns data) |
| `is_percentage` | no | `0` = nominal value, `1` = percentage |
| `page` | yes | int ≥ 1 — **pagination**: 10 periods per page, going further back in time each page (page 1 = most recent). Consecutive pages overlap by one period. Past the last page the table is empty (not an error). |
| `report_type` | yes | `1`=Income Statement, `2`=Balance Sheet, `3`=Cash Flow |
| `statement_type` | yes | `1`=Quarterly, `2`=Annual, `3`=TTM, `4`=Interim YTD, `5..8`=Q1..Q4, `9`=QoQ Growth, `10`=Quarter YoY, `11`=YTD YoY, `12`=Annual YoY, `13`=3Y CAGR |

`data: { currency, default_currency, rounding_value, data_tables: { periods, max_show_level, accounts: [{ id, level, name, values, accounts: [...], is_total_exist, is_default_expanded, max_show_level }] } }`

`values[i]` aligns with `periods[i]`; `"-"` means no data. `rounding_value` is the
unit divisor (e.g. `1000000000` = billions).

#### Example: pagination

```bash
# Page 1 - Income Statement, Annual (10 newest periods: 12M 2025, 12M 2024, ...)
curl 'http://localhost:8080/api/v1/company/BBCA/financial?data_type=1&page=1&report_type=1&statement_type=2'

# Page 2 - one period back (overlap), continuing into older periods
curl 'http://localhost:8080/api/v1/company/BBCA/financial?data_type=1&page=2&report_type=1&statement_type=2'

# Balance Sheet, Annual, as percentages
curl 'http://localhost:8080/api/v1/company/BBCA/financial?data_type=1&is_percentage=1&page=1&report_type=2&statement_type=2'

# Income Statement, TTM (periods become quarterly: Q2 2026, Q1 2026, ...)
curl 'http://localhost:8080/api/v1/company/BBCA/financial?data_type=1&page=1&report_type=1&statement_type=3'
```

**Pagination (summary):** `page` required ≥ 1 → 10 periods per page, moving further
back in time each page; consecutive pages overlap by one period; past the last page
with data is not an error (empty table). **`data_type=1` matters** - without it many
reports return an empty `periods: []`. See the param table above for valid
`report_type`/`statement_type` values.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/company/BBCA/financial?data_type=1&page=1&report_type=1&statement_type=2'
```

```json
{
  "success": true,
  "data": {
    "currency": ["IDR", "USD"],
    "default_currency": "IDR",
    "rounding_value": [1000000000, 1000000],
    "data_tables": {
      "periods": ["12M 2025", "12M 2024", "12M 2023"],
      "max_show_level": 2,
      "accounts": [
        {
          "id": 127,
          "level": 1,
          "name": "Pendapatan",
          "values": [],
          "accounts": [
            { "id": 2065, "level": 2, "name": "Pendapatan Bunga", "values": ["98,913 B"], "accounts": [], "is_total_exist": true, "is_default_expanded": false, "max_show_level": 2 }
          ],
          "is_total_exist": true,
          "is_default_expanded": false,
          "max_show_level": 2
        }
      ]
    }
  }
}
```

### `GET /api/v1/index/{symbol}/summary`
Intraday/daily price series plus per-day summary for an index (e.g. IHSG).
Proxies Stockbit's `/charts/{symbol}/daily`; the path segment is fixed to
`daily` because upstream rejects other segments (`weekly`, `monthly`, ... return
404 `Unrecognized Command`). Granularity is controlled by the `interval` query
parameter instead.

| param | required | values |
|---|---|---|
| `symbol` | path | e.g. `IHSG` |
| `from` | no | `YYYY-MM-DD`; see range rules below |
| `to` | no | `YYYY-MM-DD`; see range rules below |
| `interval` | no | pass-through, e.g. `INTERVAL_CHART_MINUTELY`; omit for daily points |

- **Range rules:** `from`/`to` must either both be provided or both omitted
  (422 `from and to must both be provided or both omitted` otherwise). When
  provided they are validated as `YYYY-MM-DD` (422 on invalid layout or
  impossible dates such as `2026-13-40`).
- **Omitting `from`/`to` defaults to the most recent trading session with
  data**: the server probes backwards from today (WIB, up to 30 days) and
  returns the latest day whose session has price points, so pre-market,
  weekend and holiday requests still return the last available session instead
  of an empty `prices[]`.
- Without `interval` the upstream returns one daily point per trading day; with
  `INTERVAL_CHART_MINUTELY` it returns the intraday minute series.
- `interval` is passed through unvalidated: upstream rejects unknown values with
  a 400 (`Your request is invalid`), which surfaces as a 500 here.
- The summary-level `change`/`percentage` fields are unreliable — the live
  capture shows `change` equal to the last price and `percentage` `"0.00"` even
  when the last point shows `-1.17%`. Prefer computing from `previous` (prior
  close) and the last `prices[]` point.

`data: { cagr, change, drawdown, markingpoint, percentage, timeframe, xaxisopt, previous, line_weight, previous_timeframe_price, chart_type, interval_in_minutes, allowed_chart_type, max_candles, prices: [{ date, formatted_date, xlabel, value, percentage, change, open, high, low, volume }] }`

`prices[]` points carry `value`/`percentage`/`formatted_date` as strings and
`change` as a number; `open`/`high`/`low`/`volume` are empty strings for the
line chart. `previous_timeframe_price` is the prior session close.

#### Example: minutely IHSG summary

```bash
curl 'http://localhost:8080/api/v1/index/IHSG/summary?from=2026-08-10&to=2026-08-10&interval=INTERVAL_CHART_MINUTELY'
```

```json
{
  "success": true,
  "data": {
    "xaxisopt": "intraday",
    "previous": 6409.654,
    "previous_timeframe_price": {
      "formatted_date": "2026-08-07",
      "value": "6409.65",
      "change": 0
    },
    "chart_type": "PRICE_CHART_TYPE_LINE",
    "prices": [
      {
        "formatted_date": "2026-08-10 09:00:00",
        "value": "6442.65",
        "percentage": "0.03",
        "change": 2.048
      },
      {
        "formatted_date": "2026-08-10 16:14:00",
        "value": "6365.37",
        "percentage": "-1.17",
        "change": -75.223
      }
    ]
  }
}
```

### `GET /api/v1/index/{symbol}/chart`
Combines the [index summary](#get-apiv1indexsymbolsummary) (`summary`) with
chartbit OHLC bars (`chart`) for the same index in one response, so a chart
page can render the intraday line and the daily candles with a single call.

| param | required | values |
|---|---|---|
| `symbol` | path | e.g. `IHSG` |
| `from` | no | `YYYY-MM-DD`; see range rules below |
| `to` | no | `YYYY-MM-DD`; see range rules below |
| `interval` | no | pass-through; affects the `summary` part only |

- Validation matches the summary endpoint: `from`/`to` must either both be
  provided or both omitted (422 `from and to must both be provided or both
  omitted` otherwise), provided dates must be `YYYY-MM-DD`, and `interval` is
  optional and passed through.
- **Omitting `from`/`to` defaults to the most recent trading session with
  data** (probed backwards from today, WIB, up to 30 days), same as the
  summary endpoint.
- **`from`/`to` are chronological** (`from` = earlier date, `to` = later one)
  and must not be reversed (422 `from must be earlier than or equal to to`).
  The summary upstream requires this order (reversed ranges return a 400), while
  chartbit daily pages backward — so the server **swaps the range for the chart
  call** internally.
- The two parts are fetched **concurrently**; if either upstream call fails the
  whole request returns 500.
- **`interval` only affects the `summary` section.** The `chart` section is
  always daily OHLC bars regardless of the requested interval granularity.
- `summary.previous`/`previous_timeframe_price` and the last `summary.prices[]`
  point should agree with the last `chart.chartbit[]` close (verified against a
  live capture: 6365.37 vs 6365.374).

`data: { summary: { ...same shape as the summary endpoint... }, chart: { allow_decimal, chartbit: [{ date, unixdate, datetime, unix_timestamp, open, high, low, close, volume, value, frequency, foreignbuy, foreignsell, foreignflow, soxclose, dividend, shareoutstanding, freq_analyzer, lot, foreign_buy, foreign_sell, symbol }] } }`

#### Example: IHSG summary + daily OHLC

```bash
curl 'http://localhost:8080/api/v1/index/IHSG/chart?from=2026-08-10&to=2026-08-10&interval=INTERVAL_CHART_MINUTELY'
```

```json
{
  "success": true,
  "data": {
    "summary": {
      "xaxisopt": "intraday",
      "previous": 6409.654,
      "previous_timeframe_price": { "formatted_date": "2026-08-07", "value": "6409.65" },
      "prices": [
        { "formatted_date": "2026-08-10 09:00:00", "value": "6442.65", "change": 2.048 },
        { "formatted_date": "2026-08-10 16:14:00", "value": "6365.37", "change": -75.223 }
      ]
    },
    "chart": {
      "allow_decimal": 0,
      "chartbit": [
        { "date": "2026-08-10", "open": 6440.597, "high": 6462.738, "low": 6362.758, "close": 6365.374, "volume": 41109487000 }
      ]
    }
  }
}
```

### `GET /api/v1/company/{symbol}/running-trade-chart`
Running trade chart: the price series plus per-broker value/volume series over a
date range or a preset period (proxies `/order-trade/running-trade/chart/{symbol}`).

| param | required | values |
|---|---|---|
| `symbol` | path | |
| `broker_code` | no | repeatable (`?broker_code=DR&broker_code=AK`); empty → upstream default set |
| `from` | no | `YYYY-MM-DD`; must be earlier than or equal to `to`; see range rules |
| `to` | no | `YYYY-MM-DD` |
| `investor_type` | no | `INVESTOR_TYPE_ALL` (default), `INVESTOR_TYPE_FOREIGN`, `INVESTOR_TYPE_DOMESTIC` |
| `market_board` | no | `BOARD_TYPE_ALL` (default), `BOARD_TYPE_REGULAR`, `BOARD_TYPE_CASH`, `BOARD_TYPE_NEGOTIATION` |
| `period` | no | `RT_PERIOD_LAST_1_DAY` (default), `RT_PERIOD_LAST_7_DAYS`, `RT_PERIOD_LAST_1_MONTH`, `RT_PERIOD_LAST_3_MONTHS`, `RT_PERIOD_YEAR_TO_DATE`, `RT_PERIOD_LAST_1_YEAR` |

- **Range rules:** `from`/`to` must either both be provided or both omitted
  (422 `from and to must both be provided or both omitted` otherwise), and when
  provided must not be reversed (422 `from must be earlier than or equal to to`
  — the upstream 400s on reversed ranges). Dates are `YYYY-MM-DD`.
- **No data for the requested range:** the upstream returns 400 when the
  requested session has no data yet (e.g. today before the market closes, or a
  future date). That is surfaced as a 422 `no running trade data for the
  requested date range`, not a 500.
- **When `from`/`to` are both omitted the `period` enum selects the timeframe**,
  defaulting to `RT_PERIOD_LAST_1_DAY` (the last 1 day, minutely points). If
  both a range and a period are supplied, the range wins.
- `broker_code` is repeatable; each value selects a broker whose series is
  included in `broker_chart_data`. Omitted → upstream picks its default set
  (live-verified: `XL, BK, AK, CC, YU` for DSSA over `2026-08-02..2026-08-11`).
- `investor_type` and `market_board` default to `INVESTOR_TYPE_ALL` /
  `BOARD_TYPE_ALL` when omitted.

`data: { from, to, data_last_updated, price_chart_data: [{ date, time, value: {raw, formatted}, datetime_label, open, high, low }], broker_chart_data: [{ type, brokers, charts: [{ broker_code, chart: [{ date, time, value, datetime_label, open, high, low }] }] }], date_session_info }`

`price_chart_data` is the symbol's price series (`open`/`high`/`low` populated).
`broker_chart_data` has one entry per series type — `TYPE_CHART_VALUE` and
`TYPE_CHART_VOLUME` — each carrying one `charts[]` entry per broker; broker chart
points only populate `value` (`open`/`high`/`low` are `null`). All `raw`/`formatted`
values are strings.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/company/DSSA/running-trade-chart?broker_code=DR&broker_code=AK&from=2026-08-02&to=2026-08-11'
```

```json
{
  "success": true,
  "data": {
    "from": "2026-08-02",
    "to": "2026-08-11",
    "data_last_updated": "2026-08-11T00:00:00Z",
    "price_chart_data": [
      {
        "date": "2026-08-03",
        "time": "00:00",
        "value": { "raw": "835", "formatted": "835" },
        "datetime_label": "03 Aug",
        "open": { "raw": "850", "formatted": "850" },
        "high": { "raw": "875", "formatted": "875" },
        "low": { "raw": "830", "formatted": "830" }
      }
    ],
    "broker_chart_data": [
      {
        "type": "TYPE_CHART_VALUE",
        "brokers": ["XL", "BK", "AK", "CC", "YU"],
        "charts": [
          {
            "broker_code": "AK",
            "chart": [
              {
                "date": "2026-08-03",
                "time": "00:00",
                "value": { "raw": "-11939190000", "formatted": "(11.9B)" },
                "datetime_label": "03 Aug",
                "open": null,
                "high": null,
                "low": null
              }
            ]
          }
        ]
      },
      {
        "type": "TYPE_CHART_VOLUME",
        "brokers": ["XL", "BK", "AK", "CC", "YU"],
        "charts": [
          {
            "broker_code": "XL",
            "chart": [
              {
                "date": "2026-08-03",
                "time": "00:00",
                "value": { "raw": "-46272", "formatted": "(46.3K)" },
                "datetime_label": "03 Aug",
                "open": null,
                "high": null,
                "low": null
              }
            ]
          }
        ]
      }
    ],
    "date_session_info": "11 Aug 2026"
  }
}
```

#### Example: last 1 day (default period)

```bash
# No from/to -> period defaults to the last 1 day (minutely points)
curl 'http://localhost:8080/api/v1/company/DSSA/running-trade-chart?broker_code=DR'

# Explicit 7-day period
curl 'http://localhost:8080/api/v1/company/DSSA/running-trade-chart?broker_code=DR&period=RT_PERIOD_LAST_7_DAYS'
```

#### Behavior notes (probed live against the upstream)

Empirically verified against `/order-trade/running-trade/chart/DSSA`:

- **Granularity follows the timeframe, not an explicit interval:** a multi-day
  range returns one point per trading day (e.g. `2026-08-02..2026-08-11` → 7
  daily points), while `from == to` or `period=RT_PERIOD_LAST_1_DAY` returns
  minutely points (e.g. 335 points for a full session `09:00..16:14`); a
  7-day `period` returns 30 points.
- **A single bound is silently ignored upstream:** `from`-only or `to`-only
  requests return the last session, dropping the given bound. This server is
  deliberately stricter and rejects them with 422
  (`from and to must both be provided or both omitted`).
- **Period beats range upstream; the reverse holds here:** when both a range and
  a `period` are sent upstream, the `period` wins (e.g. range `07-01..08-10` +
  `RT_PERIOD_LAST_7_DAYS` → last 7 days). This server sends `from`/`to` alone
  when both are provided, so the range wins on our side.
- **No-data days are not errors:** dates with no session (weekends, past
  dates) return `200` with empty `price_chart_data`/`broker_chart_data` arrays.
  The upstream only 400s when the session has no data *yet* (today before close,
  future dates) — surfaced here as a 422 (see range rules above).
- **Empty request = last session:** no params at all returns the most recent
  session's minutely data, which is what the `RT_PERIOD_LAST_1_DAY` default
  selects.
- **`broker_code` omitted → upstream default set:** a 5-broker default set
  (live-verified `XL, BK, AK, CC, YU`) is returned; it can differ per timeframe.
  An empty `broker_code=` value is accepted but yields a series with an empty
  broker code. Multi-broker responses list `charts[]` **sorted by broker code**,
  not in request order.
- **`investor_type`/`market_board` shape:** every valid enum value returns the
  same shape and point counts; empty or invalid values 400 upstream (this server
  defaults them to `ALL` and validates with `oneof`).
- **Headers do not matter:** requests succeed without `X-Platform`, `origin` or
  `referer`; `X-Platform: web`/`ios` both work.
- **Number formatting is accounting style:** negative values render as
  `raw: "-7852500", formatted: "(7.9M)"`.

### `GET /api/v1/order-trade/running-trade`
Running trade feed: the stream of executed trades for one symbol, paged by a
`trade_number` cursor (proxies `/order-trade/running-trade`).

| param | required | values |
|---|---|---|
| `symbol` | yes | one symbol (see note below) |
| `sort` | no | `ASC` (default), `DESC` |
| `order_by` | no | `RUNNING_TRADE_ORDER_BY_TIME` (default), `RUNNING_TRADE_ORDER_BY_LOT`, `RUNNING_TRADE_ORDER_BY_VALUE` |
| `limit` | no | int ≥ 1 (default 80) |
| `trade_number` | no | int64 cursor — pass the last row's `trade_number` to fetch the next page |
| `date` | no | `YYYY-MM-DD` — **omitted/empty defaults to the most recent data available** |

- **Single symbol only.** Upstream accepts `symbols[]=` (array syntax), but
  passing more than one does **not** combine feeds — it returns data for one
  symbol with inconsistent selection. Fetch per-symbol and merge client-side if
  needed.
- **Cursor paging:** response rows continue past `trade_number`; `limit` alone
  can't page past the upstream cap (verified: `limit=2000` still returns 100
  rows). Walk the feed with `trade_number` from the last row.
- `date` empty → omitted from the request, so upstream falls back to the most
  recent session with data. Invalid `YYYY-MM-DD` → `422 VALIDATION_ERROR`
  (upstream rejects malformed dates with a 400 `Invalid date`).
- Invalid `order_by` → `422 VALIDATION_ERROR` (upstream rejects unknown enum
  values with a 400 `INVALID_PARAMETER`). `sort` is lenient upstream but this
  server validates `ASC`/`DESC`.
- Upstream 400 (e.g. no data for the requested parameters) → `422`.
- `value.raw` is a **JSON number** (e.g. `630000`), not a string.

`data: { is_open_market, running_trade: [{ id, time, action, code, price,
change, lot, is_broker_exists, buyer, seller, trade_number, buyer_type,
seller_type, market_board, buy_order_number, sell_order_number,
group_order_number, value: { raw, formatted } }] }`

`action` is `buy`/`sell`; `buyer`/`seller` are `"BROKER [D|F]"` labels
(`D` = domestic, `F` = foreign); `buyer_type`/`seller_type` are
`BROKER_TYPE_LOCAL`/`BROKER_TYPE_FOREIGN`; `lot`/`price`/`change` are
display-formatted strings.

#### Example: request / response

```bash
# Latest data (date omitted -> upstream falls back to the most recent session)
curl 'http://localhost:8080/api/v1/order-trade/running-trade?symbol=BBCA&sort=ASC&order_by=RUNNING_TRADE_ORDER_BY_TIME&limit=80'

# A specific session + next page (cursor from the last row's trade_number)
curl 'http://localhost:8080/api/v1/order-trade/running-trade?symbol=BBCA&date=2026-08-13&limit=80&trade_number=31000'
```

```json
{
  "success": true,
  "data": {
    "is_open_market": false,
    "running_trade": [
      {
        "id": "4760187264",
        "time": "08:58:00",
        "action": "buy",
        "code": "BBCA",
        "price": "6,300",
        "change": "-1.18%",
        "lot": "1",
        "is_broker_exists": true,
        "buyer": "XL [D]",
        "seller": "BK [F]",
        "trade_number": "17797",
        "buyer_type": "BROKER_TYPE_LOCAL",
        "seller_type": "BROKER_TYPE_FOREIGN",
        "market_board": "RG",
        "buy_order_number": "85260",
        "sell_order_number": "13070",
        "group_order_number": "13070",
        "value": { "raw": 630000, "formatted": "630.0K" }
      }
    ]
  }
}
```

### `GET /api/v1/order-trade/order-queue`
Order queue for one symbol at one price: the pending buy/sell orders queued at
a price level, proxied straight from `/order-trade/order-queue`. Each row is a
single order waiting in the queue.

| param | required | values |
|---|---|---|
| `stock_code` | yes | one symbol, e.g. `SLIS` |
| `action_type` | no | `ACTION_TYPE_BUY`, `ACTION_TYPE_SELL`, `ACTION_TYPE_ALL` (default) |
| `board_type` | no | `BOARD_TYPE_REGULAR` (default), `BOARD_TYPE_NEGOTIATION`, `BOARD_TYPE_CASH`, `BOARD_TYPE_ALL` |
| `order_status` | no | `ORDER_STATUS_OPEN` (default), `ORDER_STATUS_FULL_MATCH`, `ORDER_STATUS_WITHDRAWN`, `ORDER_STATUS_PARTIAL_MATCH`, `ORDER_STATUS_AMEND`, `ORDER_STATUS_ALL` |
| `sort_by` | no | `SORT_BY_QUEUE` (default; the only value upstream accepts) |
| `sort_direction` | no | `SORT_DIRECTION_ASC` (default), `SORT_DIRECTION_DESC` |
| `price` | no | int — the price level queried (omitted → the whole queue) |
| `limit` | no | int ≥ 1 (default 100) |

- **One symbol per call** — `stock_code` is required; empty → `422
  VALIDATION_ERROR` (`{"stock_code":"is required"}`).
- **Enums are validated `oneof`**; unknown values (e.g. `action_type=BOGUS`) →
  `422 VALIDATION_ERROR` with a per-field message. `sort_by` only accepts
  `SORT_BY_QUEUE`.
- **Non-numeric `price`/`limit`** (e.g. `price=abc`, `limit=abc`) → `422`
  (`must be a valid integer`).
- **`pagination.has_next_page`** signals rows remain after `limit`; raise
  `limit` to fetch more in a single call.
- `price`/`open`/`lot`/`queue_lot` are **JSON numbers**; `queue_number` and
  `order_number` are **strings** (upstream serializes them as strings).
  `exchange_order_number.full` is the complete order number, `.formatted` its
  display form.
- `broker_code` is empty for orders without a public broker; `broker_group` is
  `BROKER_GROUP_{LOCAL,FOREIGN,UNSPECIFIED}`.

`data: { is_open_market, orders: [{ id, queue_number, stock_code, time,
action_type, price, status, open, lot, board_type, broker_code,
exchange_order_number: {full, formatted}, queue_lot, broker_group,
order_number }], pagination: { has_next_page } }`

- `is_open_market` = `false` outside exchange hours (live capture).
- `time` is the upstream ISO-8601 timestamp (e.g. `2026-08-14T14:09:19.282073Z`).

#### Example: request / response

```bash
# All parameters (matches the reference upstream curl)
curl 'http://localhost:8080/api/v1/order-trade/order-queue?action_type=ACTION_TYPE_ALL&board_type=BOARD_TYPE_REGULAR&limit=100&order_status=ORDER_STATUS_OPEN&price=101&sort_by=SORT_BY_QUEUE&sort_direction=SORT_DIRECTION_ASC&stock_code=SLIS'

# stock_code only -> enums/limit fall back to defaults (whole queue, any price)
curl 'http://localhost:8080/api/v1/order-trade/order-queue?stock_code=SLIS'

# Descending order + fewer rows
curl 'http://localhost:8080/api/v1/order-trade/order-queue?stock_code=SLIS&price=101&limit=5&sort_direction=SORT_DIRECTION_DESC'
```

```json
{
  "success": true,
  "data": {
    "is_open_market": false,
    "orders": [
      {
        "id": "3495619555",
        "queue_number": "1",
        "stock_code": "SLIS",
        "time": "2026-08-14T14:09:19.282073Z",
        "action_type": "ACTION_TYPE_BUY",
        "price": 101,
        "status": "ORDER_STATUS_PARTIAL_MATCH",
        "open": 39,
        "lot": 50,
        "board_type": "BOARD_TYPE_REGULAR",
        "broker_code": "YP",
        "exchange_order_number": { "full": "202608140003318022", "formatted": "3318022" },
        "queue_lot": 0,
        "broker_group": "BROKER_GROUP_FOREIGN",
        "order_number": "202608140003318022"
      }
    ],
    "pagination": { "has_next_page": true }
  }
}
```

### `GET /api/v1/order-trade/foreign-domestic/historical`
Historical foreign/domestic buy-sell aggregates for a symbol, over a period or
a date range (proxies `/order-trade/foreign-domestic/historical`).

| param | required | values |
| `symbol` | yes | one symbol (see note below) |
| `market_type` | no | `MARKET_TYPE_ALL` (default; the only value upstream accepts) |
| `period` | no | `TB_PERIOD_LAST_1_DAY` (default), `TB_PERIOD_LAST_7_DAYS`, `TB_PERIOD_LAST_1_MONTH`, `TB_PERIOD_YEAR_TO_DATE`, `TB_PERIOD_LAST_1_YEAR` |
| `from` | no | `YYYY-MM-DD`; see range rules |
| `to` | no | `YYYY-MM-DD` |

- **Range rules:** `from`/`to` must either both be provided or both omitted
  (422 `from and to must both be provided or both omitted` otherwise). A
  reversed range is rejected (422) because the upstream 400s on it
  (`The Start date must be earlier than the End date`). When both are provided
  the range **wins over `period`** (verified: `period=LAST_7_DAYS` + a month
  range returns the month range).
- **Single symbol only.** Upstream accepts a repeatable `symbols=` param but
  only the first symbol's data is returned; call per-symbol to compare.
- `market_type` only accepts `MARKET_TYPE_ALL` — any other value 400s upstream
  and is rejected here with 422. Omitted → `ALL`.
- A single bound (`from`-only or `to`-only) is silently ignored upstream
  (returns the last day) — this server rejects it with 422 instead.
- Empty `symbol` → 422; unknown symbol → upstream 404 (`Data tidak ditemukan`),
  surfaced as a 500 here.
- `historical_net[].*` raw values are **mixed JSON types upstream** (numbers
  for most fields, a string for `net_frequency`, floats for the percentages) —
  kept as-is, so consumers should handle `raw` as a number-or-string.

`data: { historical_price: [{ date, datetime_label, open/high/low/close:
{raw, formatted} }], historical_net: [{ date, datetime_label,
datetime_label_table, net_foreign, foreign_buy, foreign_sell, foreign_flow,
net_lot, net_frequency, average_price, percentage_foreign_value,
percentage_domestic_value }], last_updated, from, to }`

Each `historical_net` field is `{ raw, formatted }` (raw mixed number/string).
Rows are newest-first; `from`/`to` echo the requested or upstream-derived range.

#### Example: request / response

```bash
# Last 1 month of foreign/domestic history
curl 'http://localhost:8080/api/v1/order-trade/foreign-domestic/historical?symbol=VKTR&market_type=MARKET_TYPE_ALL&period=TB_PERIOD_LAST_1_MONTH'

# Explicit date range (wins over period)
curl 'http://localhost:8080/api/v1/order-trade/foreign-domestic/historical?symbol=VKTR&from=2026-07-01&to=2026-08-14'
```

```json
{
  "success": true,
  "data": {
    "historical_price": [
      {
        "date": "2026-08-14",
        "datetime_label": "14 Aug",
        "open": { "raw": "820", "formatted": "820" },
        "high": { "raw": "920", "formatted": "920" },
        "low": { "raw": "810", "formatted": "810" },
        "close": { "raw": "885", "formatted": "885" }
      }
    ],
    "historical_net": [
      {
        "date": "2026-08-14",
        "datetime_label": "14 Aug",
        "datetime_label_table": "14 Aug 26",
        "net_foreign": { "raw": 24004280500, "formatted": "24.00B" },
        "foreign_buy": { "raw": 45454808500, "formatted": "45.45B" },
        "foreign_sell": { "raw": 21450528000, "formatted": "21.45B" },
        "foreign_flow": { "raw": 268726637000, "formatted": "268.73B" },
        "net_lot": { "raw": 274408, "formatted": "274.41K" },
        "net_frequency": { "raw": "2060", "formatted": "2.06K" },
        "average_price": { "raw": 878, "formatted": "878" },
        "percentage_foreign_value": { "raw": 18.86, "formatted": "18.86%" },
        "percentage_domestic_value": { "raw": 81.14, "formatted": "81.14%" }
      }
    ],
    "last_updated": "14 Aug 26",
    "from": "2026-07-14",
    "to": "2026-08-14"
  }
}
```

### `GET /api/v1/company/{symbol}/orderbook`
Order book for a symbol: bid/offer price levels plus quote summary and
market-wide aggregates (proxies
`/company-price-feed/v2/orderbook/companies/{symbol}`).

| param | required | values |
|---|---|---|
| `symbol` | path | |

`data: { average, bid: [], change, close, country, domestic, down, exchange,
fbuy, fnet, foreign, frequency, fsell, high, id, lastprice, low, offer: [],
open, percentage_change, previous, status, symbol, symbol_2, symbol_3,
tradable, unchanged, up, value, volume, corp_action, notation, uma,
has_foreign_bs, iepiev, market_data, name, icon_url, ara, arb, company_type,
total_bid_offer, next_ara, next_arb, autoreject_time_left_in_sec,
auto_reject_estimation, orderbook_active_feature_mobile }`

- `bid`/`offer` are price-level lists, each `{ price, que_num, volume,
  change_percentage }` (all strings); `bid` is best-first, `offer` is
  best-first.
- `total_bid_offer`: `{ bid: { freq, lot, raw_lot, raw_freq }, offer: {...},
  bid_percent }`.
- `market_data`: `[{ label, frequency: {raw, formatted}, volume: {...},
  value: {...} }]` — one row per market (All Market, Regular, Nego, Cash).
- `ara`/`arb`/`next_ara`/`next_arb`: `{ value, visible }` price limits.
- The deep structures `iepiev`, `notation`, and `auto_reject_estimation` are
  passed through verbatim (opaque JSON) rather than re-typed.
- Numeric stats (`fbuy`, `fnet`, `value`, `volume`, ...) are JSON numbers;
  `percentage_change` is a float.
- The endpoint works with any `X-Platform` header (verified `web` and `ios`).
- Empty/missing `symbol` → `422 VALIDATION_ERROR`; upstream 400 → `422`.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/company/VKTR/orderbook'
```

```json
{
  "success": true,
  "data": {
    "average": 880,
    "bid": [
      { "price": "880", "que_num": "138", "volume": "823200", "change_percentage": "" },
      { "price": "875", "que_num": "296", "volume": "2567800", "change_percentage": "" }
    ],
    "change": 65,
    "close": 885,
    "country": "ID",
    "domestic": "81.14",
    "down": "110",
    "exchange": "IDX",
    "fbuy": 45454808500,
    "fnet": 24004280500,
    "foreign": "18.86",
    "frequency": 31496,
    "fsell": 21450528000,
    "high": 920,
    "id": "VKTR-0",
    "lastprice": 885,
    "low": 810,
    "offer": [
      { "price": "885", "que_num": "72", "volume": "497200", "change_percentage": "" }
    ],
    "open": 820,
    "percentage_change": 7.93,
    "previous": 820,
    "status": "Active",
    "symbol": "VKTR",
    "symbol_2": "VKTR",
    "symbol_3": "VKTR",
    "tradable": true,
    "unchanged": "1118",
    "up": "265",
    "value": 177331201000,
    "volume": 201417800,
    "corp_action": {
      "active": false,
      "icon": "https://assets.stockbit.com/images/corp_action_event_icon.svg",
      "text": "Perusahaan Memiliki Corporate Action"
    },
    "notation": [],
    "uma": false,
    "has_foreign_bs": true,
    "iepiev": { "symbol": "", "status": "STATUS_UNSPECIFIED" },
    "market_data": [
      {
        "label": "All Market",
        "frequency": { "raw": "31496", "formatted": "31.5 K" },
        "volume": { "raw": "201417800", "formatted": "201 M" },
        "value": { "raw": "177331201000", "formatted": "177 B" }
      }
    ],
    "name": "VKTR Teknologi Mobilitas Tbk.",
    "icon_url": "https://assets.stockbit.com/logos/companies/VKTR.png",
    "ara": { "value": "1,105", "visible": true },
    "arb": { "value": "755", "visible": true },
    "company_type": "Saham",
    "total_bid_offer": {
      "bid": { "freq": "2,762", "lot": "39,578,900", "raw_lot": "39578900", "raw_freq": "2762" },
      "offer": { "freq": "3,885", "lot": "53,346,400", "raw_lot": "53346400", "raw_freq": "3885" },
      "bid_percent": 42.6
    },
    "next_ara": { "value": "1,105", "visible": true },
    "next_arb": { "value": "755", "visible": true },
    "autoreject_time_left_in_sec": 0,
    "auto_reject_estimation": [
      { "value": 115100, "change_to_prev": { "value": 19175, "percentage": 19.99 }, "type": "AUTO_REJECT_TYPE_POSITIVE" }
    ],
    "orderbook_active_feature_mobile": "ORDER_BOOK_FEATURE_FOREIGN_BS"
  }
}
```

### `GET /api/v1/company/{symbol}/historical-summary`
Historical price summary for a symbol: one row per period over a date range,
with pagination (proxies
`/company-price-feed/historical/summary/{symbol}`).

| param | required | values |
|---|---|---|
| `symbol` | path | |
| `period` | no | `HS_PERIOD_DAILY` (default), `HS_PERIOD_WEEKLY`, `HS_PERIOD_MONTHLY` |
| `start_date` | no | `YYYY-MM-DD` |
| `end_date` | no | `YYYY-MM-DD` |
| `limit` | no | int ≥ 1 (default 50) |
| `page` | no | int ≥ 1 (default 1) |

- **Period enum** is limited to the three `HS_PERIOD_*` values above;
  anything else (e.g. `HS_PERIOD_YEARLY`) 400s upstream and is rejected here
  with 422.
- **Dates** are optional `YYYY-MM-DD`. The upstream tolerates a single bound or
  none at all, so this server does too — no both-or-neither rule for this
  endpoint.
- `limit`/`page` are integers ≥ 1; non-numeric values → 422.

`data: { result: [{ date, close, change, value, volume, frequency, foreign_buy, foreign_sell, net_foreign, open, high, low, average, change_percentage }], paginate: { next_page } }`

All numbers are plain JSON numbers (not `raw`/`formatted` objects). `paginate.next_page`
is a string cursor; follow it as `page=<next_page>` to walk the pages. The upstream
returns an empty `result` once past the end, but still echoes the next cursor, so
stop when `result` is empty.

#### Example: request / response

```bash
# Last 12 weekly bars, from the upstream curl.
curl 'http://localhost:8080/api/v1/company/DSSA/historical-summary?period=HS_PERIOD_WEEKLY&start_date=2025-08-11&end_date=2026-08-11&limit=12&page=1'
```

```json
{
  "success": true,
  "data": {
    "result": [
      {
        "date": "2026-08-10",
        "close": 945,
        "change": -30,
        "value": 1928786961000,
        "volume": 19414009,
        "frequency": 184265,
        "foreign_buy": 421517824000,
        "foreign_sell": 453168550000,
        "net_foreign": -31650726000,
        "open": 990,
        "high": 1075,
        "low": 920,
        "average": 994,
        "change_percentage": -3.08
      }
    ],
    "paginate": { "next_page": "2" }
  }
}
```

#### Behavior notes (probed live against the upstream)

Empirically verified against `/company-price-feed/historical/summary/DSSA`:

- **All params optional upstream:** omitting `period`, the date bounds, `limit`
  or `page` still returns data (defaults: the latest period, full range, 12
  rows). This server defaults `period=HS_PERIOD_DAILY`, `limit=50`, `page=1`.
- **Result is newest-first:** rows come ordered from the most recent date
  backwards (e.g. `2026-08-10` → `2026-05-25` for weekly).
- **`next_page` is a string cursor that keeps incrementing even past the end**
  (page 1000 → `"1001"` with an empty `result`); stop when `result` is empty.

### `GET /api/v1/order-trade/broker/top`
Top brokers ranked by a sort key over a period (proxies
`/order-trade/broker/top`).

| param | required | values |
|---|---|---|
| `sort` | no | `TB_SORT_BY_TOTAL_VALUE` (default), `TB_SORT_BY_NET_VALUE`, `TB_SORT_BY_BUY_VALUE`, `TB_SORT_BY_SELL_VALUE`, `TB_SORT_BY_TOTAL_FREQUENCY` |
| `order` | no | `ORDER_BY_ASC`, `ORDER_BY_DESC` (default) |
| `period` | no | `TB_PERIOD_LAST_1_DAY` (default), `TB_PERIOD_LAST_7_DAYS`, `TB_PERIOD_LAST_1_MONTH`, `TB_PERIOD_YEAR_TO_DATE` |
| `market_type` | no | `MARKET_TYPE_ALL` (default; the only accepted value upstream) |
| `eod_only` | no | boolean (`true` default); restricts to end-of-day sessions |

- **`market_type` only accepts `MARKET_TYPE_ALL`** — every other value (incl.
  `MARKET_TYPE_REGULAR`/`MARKET_TYPE_CASH`) 400s upstream and is rejected here
  with 422. Omitting it defaults to `ALL`.
- **`eod_only=false` returns fewer rows** (live-verified: 89 vs 112 for the last
  1 day) — it drops non-EOD sessions.
- All other params default when omitted; upstream tolerates an empty query.

`data: { date: { from, to, idx }, list: [{ code, name, investor_type, total_value, net_value, buy_value, sell_value, total_volume, total_frequency, group }] }`

All monetary/volume fields are **strings** (large IDR/unit amounts upstream
serializes as strings, e.g. `"3954882296950"`). `group` is `BROKER_GROUP_LOCAL` /
`BROKER_GROUP_FOREIGN`.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/order-trade/broker/top?sort=TB_SORT_BY_TOTAL_VALUE&order=ORDER_BY_DESC&period=TB_PERIOD_LAST_1_DAY&market_type=MARKET_TYPE_ALL&eod_only=true'
```

```json
{
  "success": true,
  "data": {
    "date": { "from": "2026-08-11", "to": "2026-08-11", "idx": "2026-08-11" },
    "list": [
      {
        "code": "XL",
        "name": "Stockbit Sekuritas Digital",
        "investor_type": "INVESTOR_TYPE_UNSPECIFIED",
        "total_value": "3954882296950",
        "net_value": "31002582100",
        "buy_value": "1992942439525",
        "sell_value": "1961939857425",
        "total_volume": "16818162166",
        "total_frequency": "1431582",
        "group": "BROKER_GROUP_LOCAL"
      }
    ]
  }
}
```

#### Behavior notes (probed live against the upstream)

- **All params optional upstream:** an empty query returns 200 (defaults to the
  last 1 day, sorted by total value DESC, all market types, non-EOD-only).
- **`eod_only` inverts upstream:** omitting `eod_only` upstream behaves like
  `false` (89 rows); this server defaults it to `true` (112 rows) to match the
  reference curl.

### `GET /api/v1/order-trade/broker/activity-chart`
Broker activity chart for selected symbols and brokers over a date range or
period (proxies `/order-trade/broker/activity-chart`).

| param | required | values |
|---|---|---|
| `symbols` | no | repeatable (`?symbols=BUMI&symbols=DSSA`); empty → upstream default set |
| `brokers_code` | no | repeatable (`?brokers_code=XL&brokers_code=ZP`); empty → upstream default set |
| `from` | no | `YYYY-MM-DD`; see range rules |
| `to` | no | `YYYY-MM-DD` |
| `period` | no | `RT_PERIOD_LAST_1_DAY` (default), `RT_PERIOD_LAST_7_DAYS`, `RT_PERIOD_LAST_1_MONTH`, `RT_PERIOD_LAST_3_MONTHS`, `RT_PERIOD_YEAR_TO_DATE`, `RT_PERIOD_LAST_1_YEAR` |
| `investor_type` | no | `INVESTOR_TYPE_ALL` (default), `INVESTOR_TYPE_FOREIGN`, `INVESTOR_TYPE_DOMESTIC` |
| `market_board` | no | `BOARD_TYPE_ALL` (default), `BOARD_TYPE_REGULAR`, `BOARD_TYPE_CASH`, `BOARD_TYPE_NEGOTIATION` |

- **Range rules:** `from`/`to` must either both be provided or both omitted.
  When both omitted the `period` enum selects the timeframe, defaulting to
  `RT_PERIOD_LAST_1_DAY`. If both a range and a period are supplied, the range
  wins.
- `symbols` and `brokers_code` are each repeatable; omitted → upstream picks
  its default set.

`data: { from, to, data_last_updated, chart_data: [{ type, symbols, charts: [{ symbol, chart: [{ date, time, value: {raw, formatted}, datetime_label }] }] }], date_session_info, broker_code: [], broker_name }`

`chart_data` has one entry per series type — `TYPE_CHART_VALUE` and
`TYPE_CHART_VOLUME` — each carrying one `charts[]` entry per symbol. Unlike the
running-trade point, activity-chart points only have `value` (no `open/high/low`).

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/order-trade/broker/activity-chart?period=RT_PERIOD_LAST_1_YEAR&symbols=BUMI&symbols=DSSA&brokers_code=XL&brokers_code=ZP&investor_type=INVESTOR_TYPE_ALL&market_board=BOARD_TYPE_REGULAR'
```

```json
{
  "success": true,
  "data": {
    "from": "2025-08-11",
    "to": "2026-08-11",
    "data_last_updated": "2026-08-11T00:00:00Z",
    "chart_data": [
      {
        "type": "TYPE_CHART_VALUE",
        "symbols": ["BUMI", "BRMS", "AMMN", "BMRI", "BBCA", "DSSA"],
        "charts": [
          {
            "symbol": "DSSA",
            "chart": [
              {
                "date": "2025-08-11",
                "time": "00:00",
                "value": { "raw": "10797157500", "formatted": "10.8B" },
                "datetime_label": "11 Aug"
              }
            ]
          }
        ]
      },
      {
        "type": "TYPE_CHART_VOLUME",
        "symbols": ["BUMI", "BRMS", "AMMN", "BMRI", "BBCA", "DSSA"],
        "charts": [
          {
            "symbol": "BMRI",
            "chart": [
              {
                "date": "2025-08-11",
                "time": "00:00",
                "value": { "raw": "-76880", "formatted": "(76.9K)" },
                "datetime_label": "11 Aug"
              }
            ]
          }
        ]
      }
    ],
    "date_session_info": "",
    "broker_code": ["XL", "ZP"],
    "broker_name": ""
  }
}
```

### `GET /api/v1/order-trade/broker/activity`
Broker activity transactions: per-broker buy/sell trading rows over a date range
(proxies `/order-trade/broker/activity`).

| param | required | values |
|---|---|---|
| `broker_code` | no | repeatable (`?broker_code=AK&broker_code=ZP`) |
| `transaction_type` | no | `TRANSACTION_TYPE_GROSS` (default), `TRANSACTION_TYPE_NET` |
| `investor_type` | no | `INVESTOR_TYPE_ALL` (default), `INVESTOR_TYPE_FOREIGN`, `INVESTOR_TYPE_DOMESTIC` |
| `limit` | no | int ≥ 1 (default 20) |
| `market_board` | no | `MARKET_TYPE_REGULER` (default), `MARKET_TYPE_NEGO`, `MARKET_TYPE_ALL` |
| `page` | no | int ≥ 1 (default 1) |
| `from` | no | `YYYY-MM-DD` |
| `to` | no | `YYYY-MM-DD` |
| `net_val_period` | no | `NET_VAL_PERIOD_7D` (default), `NET_VAL_PERIOD_1M`, `NET_VAL_PERIOD_3M` |

- **Enum pengejaan khas upstream:** `market_board` memakai `MARKET_TYPE_REGULER`
  (ejaan "REGULER", bukan "REGULAR") dan hanya menerima
  `REGULER`/`NEGO`/`ALL`; nilai `BOARD_TYPE_*` atau `MARKET_TYPE_REGULAR` → 400.
- **`transaction_type`** hanya `GROSS`/`NET`; `BUY`/`SELL` → 400.
- **`net_val_period`** hanya `7D`/`1M`/`3M`; `6M`/`1Y` valid enum tapi upstream
  balas 404 "Data belum tersedia untuk periode ini".
- `limit` membatasi jumlah baris per sisi (`limit=20` → 20 buy + 20 sell).

`data: { broker_activity_transaction: { brokers_buy: [], brokers_sell: [] }, from, to, broker_code, broker_name }`

Row item (flat, bukan nested):
```
{ stock_code, broker_code, type, date, value, lot, avg_price, freq,
  company_detail: { icon_url, corpaction: {active, icon, text},
                    notation: [{ notation_code, notation_desc,
                                 icon_url: {light_mode, dark_mode} }] },
  nval_trend: [{ date, nval, nvol, nfreq }] }
```
`value`/`lot`/`freq`/`nval`/`nvol`/`nfreq` are numbers (`lot`/`avg_price` bisa
fraksional); `type` is `BROKER_TYPE_LOCAL`/`BROKER_TYPE_FOREIGN`;
`broker_code` is a comma-joined string (e.g. `"AK, YU, ZP"`).

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/order-trade/broker/activity?broker_code=AK&broker_code=ZP&broker_code=YU&transaction_type=TRANSACTION_TYPE_GROSS&investor_type=INVESTOR_TYPE_ALL&limit=1&market_board=MARKET_TYPE_REGULER&page=1&from=2026-07-14&to=2026-07-31&net_val_period=NET_VAL_PERIOD_7D'
```

```json
{
  "success": true,
  "data": {
    "broker_activity_transaction": {
      "brokers_buy": [
        {
          "stock_code": "BBCA",
          "broker_code": "ZP",
          "type": "BROKER_TYPE_LOCAL",
          "date": "2026-07-14",
          "value": 4715906285000,
          "lot": 7425406,
          "avg_price": 6351.0416602135965,
          "freq": 90295,
          "company_detail": {
            "icon_url": "https://assets.stockbit.com/logos/companies/BBCA.png",
            "corpaction": { "active": false, "icon": "", "text": "" },
            "notation": []
          },
          "nval_trend": [
            { "date": "2026-08-03", "nval": 122012707500, "nvol": 193528, "nfreq": 6035 }
          ]
        }
      ],
      "brokers_sell": []
    },
    "from": "2026-07-14",
    "to": "2026-07-31",
    "broker_code": "AK, YU, ZP",
    "broker_name": ""
  }
}
```

### `GET /api/v1/order-trade/broker/activity/historical`
Per-interval broker activity: bucketed trade/price rows plus a net-value summary
grouped by `net_interval` (proxies `/order-trade/broker/activity/historical`).

| param | required | values |
|---|---|---|
| `interval` | no | `INTERVAL_DAILY` (default), `INTERVAL_WEEKLY`, `INTERVAL_MONTHLY` |
| `date_from` | no | `YYYY-MM-DD` |
| `date_to` | no | `YYYY-MM-DD` |
| `broker_codes` | no | repeatable (`?broker_codes=ZP&broker_codes=BK`) |
| `symbols` | no | repeatable (`?symbols=CUAN&symbols=ADRO`) |
| `market_board` | no | `BOARD_TYPE_ALL` (default), `BOARD_TYPE_REGULAR`, `BOARD_TYPE_CASH`, `BOARD_TYPE_NEGOTIATION` |
| `investor_type` | no | `INVESTOR_TYPE_ALL` (default), `INVESTOR_TYPE_FOREIGN`, `INVESTOR_TYPE_DOMESTIC` |
| `net_interval` | no | `INTERVAL_MONTHLY` (default), `INTERVAL_DAILY`, `INTERVAL_WEEKLY` |

- **Enum beda dari endpoint activity lain:** di sini `market_board` memakai
  `BOARD_TYPE_*` (bukan `MARKET_TYPE_*`) dan `interval`/`net_interval` memakai
  `INTERVAL_DAILY/WEEKLY/MONTHLY`.
- `date_to` dicap ke tanggal tersedia terakhir di upstream (mis. minta sampai
  `2026-08-31` → response `date_to: "2026-08-12"`).
- `records` berisi satu bucket per `interval`; saat memfilter satu broker,
  `broker_code` terisi per record, selain itu kosong (aggregat semua broker).
- `summary` menggabungkan net-value per `net_interval`; upstream membalas
  `summary.group_type` dengan `INTERVAL_TYPE_*` (mis. `INTERVAL_TYPE_MONTHLY`)
  walau `net_interval` diminta sebagai `INTERVAL_MONTHLY`.

`data: { date_from, date_to, symbols: [], broker_codes: [], broker_name, records: [], pagination, summary }`

Record item:
```
{ date, broker_code, trade_activity: {
    net_summary:     { avg_price, freq, lot, value },
    buy_summary:     { avg_price, freq, lot, value },
    sell_summary:    { avg_price, freq, lot, value },
    foreign_summary: { foreign_buy, foreign_sell, net_foreign },
    total_buy_lot:   { amount, pct },
    total_sell_lot:  { amount, pct } },
  price_activity: {
    close_price: "870",             // string
    return_summary: { amount, pct } } }
```
`pagination: { page, limit, has_next, has_prev }`;
`summary: { group_type, data: [{ date_from, date_to, net_summary }] }`.
All value/lot/freq/amount/pct fields are numbers (`lot`/`avg_price` bisa
fraksional); `close_price` is a string.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/order-trade/broker/activity/historical?interval=INTERVAL_DAILY&date_from=2026-07-01&date_to=2026-08-31&broker_codes=ZP&broker_codes=BK&symbols=CUAN&market_board=BOARD_TYPE_REGULAR&investor_type=INVESTOR_TYPE_ALL&net_interval=INTERVAL_MONTHLY'
```

```json
{
  "success": true,
  "data": {
    "date_from": "2026-07-01",
    "date_to": "2026-08-12",
    "symbols": ["CUAN"],
    "broker_codes": ["ZP", "BK"],
    "broker_name": "",
    "records": [
      {
        "date": "2026-08-12",
        "broker_code": "",
        "trade_activity": {
          "net_summary": { "avg_price": 796.78, "freq": 4947, "lot": -141664, "value": -11740235500 },
          "buy_summary": { "avg_price": 786.08, "freq": 1937, "lot": 422964, "value": 33248405500 },
          "sell_summary": { "avg_price": 796.78, "freq": 4947, "lot": 564628, "value": 44988641000 },
          "foreign_summary": { "foreign_buy": 0, "foreign_sell": 0, "net_foreign": 0 },
          "total_buy_lot": { "amount": 422964, "pct": 42.83 },
          "total_sell_lot": { "amount": 564628, "pct": 57.17 }
        },
        "price_activity": {
          "close_price": "870",
          "return_summary": { "amount": 73.22, "pct": 8.42 }
        }
      }
    ],
    "pagination": { "page": 1, "limit": 100, "has_next": false, "has_prev": false },
    "summary": {
      "group_type": "INTERVAL_TYPE_MONTHLY",
      "data": [
        { "date_from": "2026-08-01", "date_to": "2026-08-12", "net_summary": { "avg_price": 731.67, "freq": 13735, "lot": -514137, "value": -36687700500 } },
        { "date_from": "2026-07-01", "date_to": "2026-07-31", "net_summary": { "avg_price": 676.19, "freq": 24248, "lot": 7726, "value": -1823597000 } }
      ]
    }
  }
}
```

## Stream

### `GET /api/v1/user/{username}/stream`
A user's stream feed (social posts / stock reports), paged by cursor
(proxies `/stream/v3/user/{username}`).

| param | required | values |
|---|---|---|
| `username` | path | |
| `category` | no | `STREAM_CATEGORY_MAIN_IDEAS` (default), `STREAM_CATEGORY_NEWS` |
| `last_stream_id` | no | int64 cursor — pass the previous response's `pagination.next_cursor` to fetch older posts (default 0) |
| `limit` | no | int ≥ 1 (default 20) |

- `category` defaults to `STREAM_CATEGORY_MAIN_IDEAS` when omitted; `limit` to `20`.
- Non-numeric `last_stream_id`/`limit`, `limit=0`, or an invalid `category` →
  `422 VALIDATION_ERROR`.
- Upstream returns 400 when no data exists for the requested parameters
  (`no user stream data for the requested parameters`), surfaced here as `422`.
- `stream[].title_url` is a relative path such as `streams/announcement/<uuid>`;
  resolve it against the [announcement endpoint](#get-apiv1streamannouncementstream_id)
  to fetch the announcement attachments for that post.

`data: { stream: [{ stream_id, title_url, title, content, created_at, created_display,
updated_at, user: {...}, status: {...}, total_replies, total_likes, type,
parent_stream_id, reports: [], topics: [], summary: {...} | null,
reaction: {...} | null }], pagination: { is_last_page, next_cursor, total },
invalid_watchlist_ids: [] }`

- `user`: `{ user_id, is_author, username, fullname, avatar, is_verified,
  user_privilege, is_pro, country, verified_status }`
- `status`: `{ is_pinned, is_trending, is_reposted, is_liked, is_saved, is_followed,
  is_unavailable, is_junk, is_spam, is_violation, is_deleted }` (booleans)
- `reports`: `[{ type }]`; `topics`: `[string]`
- `summary` (`null` when absent): `{ title, summary, key_points, key_takeaway,
  model, model_version }`
- `reaction` (`null` when absent): `{ reactions: [{ reaction, total }], total,
  my_reaction }`
- `type` is `STREAM_TYPE_REPORT` / `STREAM_TYPE_POST`; `pagination.next_cursor` is
  the `last_stream_id` for the next page.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/user/StockbitReports/stream?category=STREAM_CATEGORY_MAIN_IDEAS&last_stream_id=34884487&limit=20'
```

```json
{
  "success": true,
  "data": {
    "stream": [
      {
        "stream_id": 34884767,
        "title_url": "streams/announcement/41a33437efb2c797eb3a8e9055bb318d",
        "title": "Laporan Harian atas Nilai Aktiva Bersih dan Komposisi Portofolio [XKMS]",
        "content": "",
        "created_at": "2026-08-14 18:17:44",
        "created_display": "14 Aug 26, 18:17",
        "updated_at": "0000-00-00 00:00:00",
        "user": {
          "user_id": 3,
          "is_author": false,
          "username": "StockbitReports",
          "fullname": "Stockbit Reports",
          "avatar": "https://avatar.stockbit.com/3-1366883879.jpeg",
          "is_verified": true,
          "user_privilege": "PRIVILEGE_MEMBER",
          "is_pro": false,
          "country": "COUNTRY_ID",
          "verified_status": "VERIFIED_STATUS_COMMUNITY"
        },
        "status": {
          "is_pinned": false,
          "is_trending": false,
          "is_reposted": false,
          "is_liked": false,
          "is_saved": false,
          "is_followed": false,
          "is_unavailable": false,
          "is_junk": false,
          "is_spam": false,
          "is_violation": false,
          "is_deleted": false
        },
        "total_replies": 0,
        "total_likes": 0,
        "type": "STREAM_TYPE_REPORT",
        "parent_stream_id": 0,
        "reports": [{ "type": "Others" }],
        "topics": ["XKMS"],
        "summary": {
          "title": "Laporan Harian Nilai Aset Bersih",
          "summary": "XKMS melaporkan nilai aset bersih ...",
          "key_points": ["Nilai aset bersih tercatat Rp126,71 miliar."],
          "key_takeaway": "Laporan harian ini bersifat rutin.",
          "model": "SUMMARY_MODEL_AI",
          "model_version": "v1"
        },
        "reaction": null
      }
    ],
    "pagination": { "is_last_page": false, "next_cursor": 34884487, "total": 20 },
    "invalid_watchlist_ids": []
  }
}
```

### `GET /api/v1/stream/announcement/{stream_id}`
Announcement attachments (e.g. PDF reports) published on a stream post
(proxies `/stream/announcement/{id}`). Non-report posts return an empty list.

| param | required | values |
|---|---|---|
| `stream_id` | path | stream UUID — the last segment of a post's `title_url` |

- Empty `stream_id` → `422 VALIDATION_ERROR` (`{"stream_id":"is required"}`).
  Via the router, an empty segment doesn't match the route and falls through to
  `404` instead.
- An unrecognised `stream_id` (upstream 400) → `422 VALIDATION_ERROR`
  (`no announcement data for the requested stream`).
- Each item is one attached file. One announcement is often split into a base
  file plus `_lampN` (attachment) files sharing the same `headline`.

`data: [{ id, company_id, posted_on, headline, title, attachment, retrieved_on,
symbol, name, company_icon_url }]`

- `id` is the attachment id (not the stream id); `title` is the file name;
  `attachment` is the fully-qualified PDF URL.
- `posted_on`/`retrieved_on` are `YYYY-MM-DD HH:MM:SS` strings.
- `symbol`/`name` identify the issuing company; empty list responses are
  `data: []`.

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/stream/announcement/f3e83a0aeb3c9c48800b7f3beafc8aba'
```

```json
{
  "success": true,
  "data": [
    {
      "id": 3547541,
      "company_id": 497,
      "posted_on": "2026-08-15 07:41:51",
      "headline": "Rencana Transaksi Material Dengan Persetujuan RUPS (KOREKSI) [SILO]",
      "title": "f-32120989-0_SILO_Rencana_Transaksi_Material_Dengan_Persetujuan_RUPS_32120989.pdf",
      "attachment": "https://emitten-announcement.stockbit.com/attachments/f-32120989-0_SILO_Rencana_Transaksi_Material_Dengan_Persetujuan_RUPS_32120989.pdf",
      "retrieved_on": "2026-08-15 00:50:17",
      "symbol": "SILO",
      "name": "Siloam International Hospitals Tbk",
      "company_icon_url": "https://assets.stockbit.com/logos/companies/SILO.png"
    }
  ]
}
```

### `GET /api/v1/stream/conversation/{stream_id}`
A stream post plus one page of its replies — the content behind a
[notification](#get-apiv1notifications) item (proxies
`/stream/v3/conversation/{id}`; upstream exposes it as a body-less POST, this
server accepts a plain GET).

| param | required | values |
|---|---|---|
| `stream_id` | path | the post id — a notification's `link_to.value` |

- **Chaining:** `GET /api/v1/notifications` → take `items[].link_to.value` →
  pass it here to render the notification's content. For report posts,
  `parent.title_url` (`streams/announcement/<uuid>`) feeds the
  [announcement endpoint](#get-apiv1streamannouncementstream_id) to fetch the
  attached PDFs.
- Unknown/expired `stream_id` (upstream 400/404) → `422 VALIDATION_ERROR`
  (`no conversation data for the requested stream`). Via the router, an empty
  segment doesn't match the route and falls through to `404`.
- `parent` is the requested post; `replies[]` are its comments
  (`STREAM_TYPE_POST`). Both use the same post shape as the
  [user stream](#get-apiv1userusernamestream).
- `more: true` means more replies exist server-side; paging is not exposed yet.
- `parent.summary` is the AI summary (`null` when absent); `reaction.reactions`
  aggregates emoji counts.

`data: { more, prev, parent: {post}, replies: [{post}] }`

#### Example: request / response

```bash
# Resolve a notification's content (id from link_to.value)
curl 'http://localhost:8080/api/v1/stream/conversation/35071377'
```

```json
{
  "success": true,
  "data": {
    "more": false,
    "prev": true,
    "parent": {
      "stream_id": 35071377,
      "title_url": "streams/announcement/1c0bec4af147edd99368381d57767446",
      "title": "Penyampaian Laporan Keuangan Interim Yang Ditelaah Secara Terbatas [BRIS]",
      "created_display": "21 Aug 26, 17:18",
      "user": { "username": "StockbitReports", "fullname": "Stockbit Reports", "is_verified": true },
      "total_replies": 24,
      "total_likes": 57,
      "type": "STREAM_TYPE_REPORT",
      "topics": ["BRIS"],
      "summary": {
        "summary": "BRIS (Bank Syariah Indonesia) membukukan laba bersih H1 2026 sebesar Rp4,16 triliun, naik 11,08% YoY ...",
        "key_points": ["Laba bersih semester I 2026 mencapai Rp4,16 triliun, tumbuh 11,08% YoY."],
        "model": "SUMMARY_MODEL_AI"
      },
      "reaction": { "reactions": [{ "reaction": "👍", "total": 55 }], "total": 57 }
    },
    "replies": [
      {
        "stream_id": 35071473,
        "content": "good luck",
        "user": { "username": "zackmoy", "fullname": "Achmad zaqi mubarok" },
        "type": "STREAM_TYPE_POST"
      }
    ]
  }
}
```

### `GET /api/v1/search`
Global symbol/entity search (proxies `/search`). Returns matching companies,
insider labels, chat rooms, people, sectors and industries in one payload.

| param | required | values |
|---|---|---|
| `keyword` | yes | search text, e.g. `BBRI` |
| `page` | no | int ≥ 1 (default 1) |
| `type` | no | `company` (default), `insider`, `people`, `sector`, `industries`, `chat`, `all` |

- Missing `keyword` → `422 VALIDATION_ERROR` (`{"keyword":"is required"}`).
- Invalid `page` (non-numeric or < 1) or unknown `type` → `422 VALIDATION_ERROR`.
- Upstream rejects empty `type` with a 400 → mapped to `422 VALIDATION_ERROR`
  (`invalid search parameters`).
- `data.pagination` only exposes `has_more_*` flags (no page/total counts).
- Empty sections are `[]` (never `null`).

`data: { chat, company, insider, people, sector, industries, pagination }` — each
section item carries an `id` plus type-specific fields (`name`/`label`,
`is_verified`, `is_following`, `is_tradeable`/`tradeable`, `icon_url`, …).

#### Example: request / response

```bash
curl 'http://localhost:8080/api/v1/search?keyword=BBRI&page=1&type=company'
```

```json
{
  "success": true,
  "data": {
    "chat": [],
    "company": [
      {
        "id": "59",
        "name": "BBRI",
        "country": "ID",
        "desc": "Bank Rakyat Indonesia (Persero) Tbk.",
        "exchange": "IDX",
        "is_following": false,
        "img": "https://assets.stockbit.com/logos/companies/BBRI.png",
        "is_verified": false,
        "other": "PT Bank Rakyat Indonesia (Persero) Tbk",
        "status": "ACTIVE",
        "symbol_2": "",
        "symbol_3": "",
        "total_followers": 107291,
        "is_tradeable": true,
        "type": "Saham",
        "url": "symbol/BBRI",
        "icon_url": "https://assets.stockbit.com/logos/companies/BBRI.png"
      }
    ],
    "insider": [],
    "people": [],
    "sector": [],
    "industries": [],
    "pagination": {
      "has_more_companies": false,
      "has_more_insiders": false,
      "has_more_users": false
    }
  }
}
```

## Notifications

### `GET /api/v1/notifications`
The authenticated account's notification feed (proxies `/notification`). The
server's own credentials identify the user — no client `Authorization` needed.

| param | required | values |
|---|---|---|
| `last_id` | no | int64 cursor — pass the last row's `id` to fetch older items (default 0 = newest) |
| `limit` | no | int 1..25 (default 20; upstream caps at 25) |
| `types` | no | repeatable (`?types=A&types=B`); omitted → all types |

- **Type filter** accepts the seven reference types: `NOTIF_TYPE_NEW_REPORT`,
  `NOTIF_TYPE_NEWSFEED`, `NOTIF_TYPE_COMPANY_PUBLIC_EXPOSE`,
  `NOTIF_TYPE_COMPANY_SHAREHOLDING`, `NOTIF_TYPE_COMPANY_DIVIDEND`,
  `NOTIF_TYPE_COMPANY_CORP_ACTION`, `NOTIF_TYPE_COMPANY_OTHERS`. Unknown values →
  `422 VALIDATION_ERROR`.
- **Response can contain other types** (e.g. `NOTIF_TYPE_POSTALERT`) when
  `types` is omitted — upstream returns every kind for the account.
- Non-numeric `last_id`/`limit` → `422`; `last_id=0` is omitted from the
  upstream call (same result as sending it).
- `message` is an upstream template with `%key%` placeholders; resolve them with
  the matching `masks[]` entry (`key` → `text`), rendered bold via `tag`.

`data: { unread, items: [{ id, type, avatar, message, masks: [{ key, text,
tag, type }], link_to: { key, value }, created, is_read }] }`

- `unread` is the account-wide unread count (not affected by filters).
- `link_to.value` identifies the related object (e.g. stream/report id);
  `link_to.key` its kind.
- `created` is an RFC3339 UTC timestamp.
- Items are newest-first.

#### Example: request / response

```bash
# The seven-type filter from the reference upstream curl
curl 'http://localhost:8080/api/v1/notifications?last_id=0&limit=20&types=NOTIF_TYPE_NEW_REPORT&types=NOTIF_TYPE_NEWSFEED&types=NOTIF_TYPE_COMPANY_PUBLIC_EXPOSE&types=NOTIF_TYPE_COMPANY_SHAREHOLDING&types=NOTIF_TYPE_COMPANY_DIVIDEND&types=NOTIF_TYPE_COMPANY_CORP_ACTION&types=NOTIF_TYPE_COMPANY_OTHERS'

# Defaults: newest page of everything
curl 'http://localhost:8080/api/v1/notifications'
```

```json
{
  "success": true,
  "data": {
    "unread": 0,
    "items": [
      {
        "id": 85348738,
        "type": "NOTIF_TYPE_COMPANY_OTHERS",
        "avatar": "",
        "message": "[%0kutu%] Report: \"Hadir di IIGCE 2026, ... [PGEO]\"",
        "masks": [
          { "key": "0kutu", "text": "PGEO", "tag": "PAYLOAD_MASK_TAG_BOLD", "type": "PAYLOAD_MASK_TYPE_COMPANY" }
        ],
        "link_to": { "key": 1, "value": "35074794" },
        "created": "2026-08-21T14:10:53Z",
        "is_read": true
      }
    ]
  }
}
```

## Notes

- `symbol` path params are validated as `required` (422 on empty). Query enums are
  validated with `oneof`; invalid values → `422 VALIDATION_ERROR`.
- Upstream rate-limits aggressively when looping; keep a delay between calls when
  fetching many symbols.
- The stockbit client sends mobile headers (`X-Platform: iOS`) by default; some
  mobile-only endpoints (e.g. `/financial`) return empty data without them.