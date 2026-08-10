# REST API Reference

All endpoints return a single response envelope. Every `/v1/*` route proxies the
Stockbit (Exodus) API using the server's own credentials — no client `Authorization`
header is needed.

## Endpoint index

All routes registered in `internal/delivery/http/router.go` are documented below:

| # | endpoint | section |
|---|---|---|
| 1 | `GET /health` | [Health & ops](#get-health) |
| 2 | `GET /v1/trending` | [Market data](#get-v1trending) |
| 3 | `GET /v1/market-mover` | [Market data](#get-v1market-mover) |
| 4 | `GET /v1/market-session` | [Market data](#get-v1market-session) |
| 5 | `GET /v1/indexes` | [Market data](#get-v1indexes) |
| 6 | `GET /v1/sectors` | [Market data](#get-v1sectors) |
| 7 | `GET /v1/stocks` | [Market data](#get-v1stocks) |
| 8 | `GET /v1/company/{symbol}/profile` | [Company fundamentals](#get-v1companysymbolprofile) |
| 9 | `GET /v1/company/{symbol}/subsidiaries` | [Company fundamentals](#get-v1companysymbolsubsidiaries) |
| 10 | `GET /v1/company/{symbol}/shareholding-composition` | [Company fundamentals](#get-v1companysymbolshareholding-composition) |
| 11 | `GET /v1/insider/shareholding-network` | [Company fundamentals](#get-v1insidershareholding-network) |
| 12 | `GET /v1/insider/majorholder` | [Company fundamentals](#get-v1insidermajorholder) |
| 13 | `GET /v1/market-detector/{symbol}` | [Market detector](#get-v1market-detectorsymbol) |
| 14 | `GET /v1/top-stock` | [Top stock](#get-v1top-stock) |
| 15 | `GET /v1/company/{symbol}/corp-actions` | [Company fundamentals](#get-v1companysymbolcorp-actions) |
| 16 | `GET /v1/company/{symbol}/keystats` | [Company fundamentals](#get-v1companysymbolkeystats) |
| 17 | `GET /v1/company/{symbol}/price-performance` | [Company fundamentals](#get-v1companysymbolprice-performance) |
| 18 | `GET /v1/company/{symbol}/chart` | [Company fundamentals](#get-v1companysymbolchart) |
| 19 | `GET /v1/company/{symbol}/fundachart` | [Company fundamentals](#get-v1companysymbolfundachart) |
| 20 | `GET /v1/fundachart/metrics` | [Company fundamentals](#get-v1fundachartmetrics) |
| 21 | `GET /v1/company/{symbol}/financial` | [Company fundamentals](#get-v1companysymbolfinancial) |

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

### `GET /v1/trending`
Top trending stocks. `data: [{ symbol, name, last, change, percent, previous, logo, status }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/trending'
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

### `GET /v1/market-mover`
Market movers by type.

| param | required | values |
|---|---|---|
| `mover_type` | yes | `MOVER_TYPE_TOP_GAINER`, `MOVER_TYPE_TOP_LOSER`, `MOVER_TYPE_TOP_VALUE`, `MOVER_TYPE_TOP_VOLUME`, `MOVER_TYPE_TOP_FREQUENCY`, `MOVER_TYPE_NET_FOREIGN_BUY`, `MOVER_TYPE_NET_FOREIGN_SELL`, `MOVER_TYPE_IEVAL_TOP_GAINER` |
| `filter_stocks` | no | repeatable; `FILTER_STOCKS_TYPE_{MAIN,DEVELOPMENT,ACCELERATION,NEW_ECONOMY,SPECIAL_MONITORING}_BOARD`, `FILTER_STOCKS_TYPE_WARRANT_AND_RIGHT` |

`data: [{ symbol, name, price, change_value, change_percent, value, volume, freq, net_foreign_buy, net_foreign_sell, iep, iev, ieval, iep_change_prev }]`

#### Example: board filters

```bash
# Top gainers on the main board
curl 'http://localhost:8080/v1/market-mover?mover_type=MOVER_TYPE_TOP_GAINER&filter_stocks=FILTER_STOCKS_TYPE_MAIN_BOARD'

# Net foreign buy across all boards (filter optional, repeatable)
curl 'http://localhost:8080/v1/market-mover?mover_type=MOVER_TYPE_NET_FOREIGN_BUY'
curl 'http://localhost:8080/v1/market-mover?mover_type=MOVER_TYPE_TOP_VALUE&filter_stocks=FILTER_STOCKS_TYPE_MAIN_BOARD&filter_stocks=FILTER_STOCKS_TYPE_DEVELOPMENT_BOARD'
```

- `filter_stocks` can be repeated (`?filter_stocks=A&filter_stocks=B`) to combine boards.

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/market-mover?mover_type=MOVER_TYPE_TOP_GAINER'
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

### `GET /v1/market-session`
Current / upcoming market session. `data: { datetime, fca, regular }` — `fca` and
`regular` are session states with `{ state_name, is_last_session, is_end_of_day,
state_start_time, state_end_time, time_left }`.

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/market-session'
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

### `GET /v1/indexes`
IDX index list. `data: { main: [{symbol,name,last,change,percent,marketcap}], all: [...] }`

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/indexes'
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

### `GET /v1/sectors`
Sector indexes with nested constituent companies.

`data: [{ symbol, icon, type, last, change, percent, companies: [{ symbol, name, last, change, percent, volume, value, marketcap, icon_url, company_status, is_uma }] }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/sectors'
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

### `GET /v1/stocks`
IHSG constituent list. `data: [{ symbol, name, last, change, percent, volume, value, marketcap, icon_url, company_status, is_uma }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/stocks'
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

### `GET /v1/company/{symbol}/profile`
Company profile. `symbol` is path param (required, uppercase as traded).

`data: { background, history, key_executive, address, subsidiary, beneficiary, shareholder, shareholder_director_commissioner, shareholder_numbers, shareholder_one_percent }`

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/company/BBCA/profile'
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

### `GET /v1/company/{symbol}/subsidiaries`
Subsidiary list.

`data: { currency, last_updated_period, unit, subsidiaries: [{ company_name, business_type, location, commercial_year, total_assets, percentage, operational_status, period, raw }] }`

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/company/BBCA/subsidiaries'
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

### `GET /v1/company/{symbol}/shareholding-composition`
Insider shareholding composition per reporting period.

| param | required | notes |
|---|---|---|
| `symbol` | path | |
| `period_start` | no | `YYYY-MM-DD`, filters upstream periods |
| `period_end` | no | `YYYY-MM-DD` |

`data: [{ report_date, total_shares: {raw, formatted}, compositions: [{ label, shares, percentage, colors }] }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/company/BBCA/shareholding-composition'
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

### `GET /v1/insider/shareholding-network`
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
curl 'http://localhost:8080/v1/insider/shareholding-network?root_id=12345&root_type=SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR'

# Cap graph depth and branches per node (optional)
curl 'http://localhost:8080/v1/insider/shareholding-network?root_id=12345&root_type=SHAREHOLDING_NETWORK_NODE_TYPE_COMPANY&max_depth=3&max_edge_per_node=10'
```

- Empty `root_id` → `422 VALIDATION_ERROR` (`{"root_id":"is required"}`). `root_type`
  is only validated as required server-side; pass a supported node type or the
  upstream call may fail.

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/insider/shareholding-network?root_id=54&root_type=SHAREHOLDING_NETWORK_NODE_TYPE_COMPANY'
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

### `GET /v1/insider/majorholder`
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
curl 'http://localhost:8080/v1/insider/majorholder?symbols=BBCA,BBRI&page=1&limit=10'

# Filter by action type + data source + paging
curl 'http://localhost:8080/v1/insider/majorholder?symbols=BBCA&action_type=ACTION_TYPE_BUY&source_type=SOURCE_TYPE_KSEI&page=2&limit=20'
```

- `page`/`limit` default to upstream values (limit 20); `is_more` indicates whether
  more pages exist.

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/insider/majorholder?symbols=BBCA&page=1&limit=2'
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

### `GET /v1/market-detector/{symbol}`
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
curl 'http://localhost:8080/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10&limit=25'
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

### `GET /v1/top-stock`
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
curl 'http://localhost:8080/v1/top-stock?start=2026-08-09&end=2026-08-10&limit=25'
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

### `GET /v1/company/{symbol}/corp-actions`
Corporate action history. `action_info` is a typed payload dispatched by `action_type`.

| param | required | notes |
|---|---|---|
| `symbol` | path | |
| `limit` | no | int |

`data: [{ action_type, action_info: { rups | rightissue | stocksplit | <other>: {...} } }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/company/BBCA/corp-actions'
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

### `GET /v1/company/{symbol}/keystats`
Key-stat ratios (display-formatted strings, ~10 years).

| param | required | notes |
|---|---|---|
| `symbol` | path | |
| `year_limit` | no | int |

`data: { closure_fin_items_results, financial_year_parent, stats, info, dividend_group, financial_report_currency }`

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/company/BBCA/keystats'
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

### `GET /v1/company/{symbol}/price-performance`
Price performance per timeframe. `data: { prices: [{ close, high, low, percentage, timeframe }] }`
(timeframes: `1D 1W 1M 3M 6M YTD 1Y`).

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/company/BBCA/price-performance'
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

### `GET /v1/company/{symbol}/chart`
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
curl 'http://localhost:8080/v1/company/BBCA/chart?timeframe=daily&from=2026-08-07&to=2026-07-07'
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
| 1M | `2026-08-07` | `2026-07-07` | `curl 'http://localhost:8080/v1/company/BBCA/chart?timeframe=daily&from=2026-08-07&to=2026-07-07'` |
| 3M | `2026-08-07` | `2026-05-07` | `curl 'http://localhost:8080/v1/company/BBCA/chart?timeframe=daily&from=2026-08-07&to=2026-05-07'` |
| 6M | `2026-08-07` | `2026-02-07` | `curl 'http://localhost:8080/v1/company/BBCA/chart?timeframe=daily&from=2026-08-07&to=2026-02-07'` |
| 1Y | `2026-08-07` | `2025-08-07` | `curl 'http://localhost:8080/v1/company/BBCA/chart?timeframe=daily&from=2026-08-07&to=2025-08-07'` |

To generate the `to` date dynamically from a client:

```bash
# GNU date (Linux)
TO=$(date -d '3 months ago' +%F)
FROM=$(date +%F)
curl "http://localhost:8080/v1/company/BBCA/chart?timeframe=daily&from=$FROM&to=$TO"

# BSD date (macOS)
TO=$(date -v-3m +%F)
FROM=$(date +%F)
curl "http://localhost:8080/v1/company/BBCA/chart?timeframe=daily&from=$FROM&to=$TO"
```

> If `date +%F` lands on a weekend/holiday (non-trading day), use the most
> recent trading day as `from` instead — e.g. fall back to the previous Friday
> — since a non-trading-day `from` is untested upstream behavior and may return
> an empty or partial `chartbit`.

### `GET /v1/company/{symbol}/fundachart`
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
curl 'http://localhost:8080/v1/company/BBCA/fundachart?item=2661,2525&timeframe=5y'

# List available item_id values
curl 'http://localhost:8080/v1/fundachart/metrics?metric_name=fundachart'
```

- `timeframe` optional (`1y 3y 5y 10y`, default `10y`); `item` required.

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/company/BBCA/fundachart?item=2661&timeframe=5y'
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

### `GET /v1/fundachart/metrics`
Catalog of available `item` ids (recursive 3-level tree) to use with `/fundachart`.

| param | required | values |
|---|---|---|
| `metric_name` | yes | `fundachart` |

`data: [{ fitem_id, fitem_name, show_chart_icon, child: [...] }]`

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/fundachart/metrics?metric_name=fundachart'
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

### `GET /v1/company/{symbol}/financial`
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
curl 'http://localhost:8080/v1/company/BBCA/financial?data_type=1&page=1&report_type=1&statement_type=2'

# Page 2 - one period back (overlap), continuing into older periods
curl 'http://localhost:8080/v1/company/BBCA/financial?data_type=1&page=2&report_type=1&statement_type=2'

# Balance Sheet, Annual, as percentages
curl 'http://localhost:8080/v1/company/BBCA/financial?data_type=1&is_percentage=1&page=1&report_type=2&statement_type=2'

# Income Statement, TTM (periods become quarterly: Q2 2026, Q1 2026, ...)
curl 'http://localhost:8080/v1/company/BBCA/financial?data_type=1&page=1&report_type=1&statement_type=3'
```

**Pagination (summary):** `page` required ≥ 1 → 10 periods per page, moving further
back in time each page; consecutive pages overlap by one period; past the last page
with data is not an error (empty table). **`data_type=1` matters** - without it many
reports return an empty `periods: []`. See the param table above for valid
`report_type`/`statement_type` values.

#### Example: request / response

```bash
curl 'http://localhost:8080/v1/company/BBCA/financial?data_type=1&page=1&report_type=1&statement_type=2'
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

## Notes

- `symbol` path params are validated as `required` (422 on empty). Query enums are
  validated with `oneof`; invalid values → `422 VALIDATION_ERROR`.
- Upstream rate-limits aggressively when looping; keep a delay between calls when
  fetching many symbols.
- The stockbit client sends mobile headers (`X-Platform: iOS`) by default; some
  mobile-only endpoints (e.g. `/financial`) return empty data without them.