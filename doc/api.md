# REST API Reference

All endpoints return a single response envelope. Every `/v1/*` route proxies the
Stockbit (Exodus) API using the server's own credentials — no client `Authorization`
header is needed.

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

- `200` success → `success: true`
- `422` invalid input → `error.code: VALIDATION_ERROR` with per-field `details`
- `429` rate limit → `TOO_MANY_REQUESTS` + `Retry-After` header
- `500` upstream/app failure → `INTERNAL_ERROR`

## Market data

### `GET /health`
Liveness + DB/Redis connectivity.

### `GET /v1/trending`
Top trending stocks. `data: [{ symbol, name, last, change, percent, previous, logo, status }]`

### `GET /v1/market-mover`
Market movers by type.

| param | required | values |
|---|---|---|
| `mover_type` | yes | `MOVER_TYPE_TOP_GAINER`, `MOVER_TYPE_TOP_LOSER`, `MOVER_TYPE_TOP_VALUE`, `MOVER_TYPE_TOP_VOLUME`, `MOVER_TYPE_TOP_FREQUENCY`, `MOVER_TYPE_NET_FOREIGN_BUY`, `MOVER_TYPE_NET_FOREIGN_SELL`, `MOVER_TYPE_IEVAL_TOP_GAINER` |
| `filter_stocks` | no | repeatable; `FILTER_STOCKS_TYPE_{MAIN,DEVELOPMENT,ACCELERATION,NEW_ECONOMY,SPECIAL_MONITORING}_BOARD`, `FILTER_STOCKS_TYPE_WARRANT_AND_RIGHT` |

`data: [{ symbol, name, price, change_value, change_percent, value, volume, freq, net_foreign_buy, net_foreign_sell, iep, iev, ieval, iep_change_prev }]`

### `GET /v1/market-session`
Current / upcoming market session. `data: { market_session_datetime, segments: [...] }`

### `GET /v1/indexes`
IDX index list. `data: { main: [{symbol,name,last,change,percent,marketcap}], all: [...] }`

### `GET /v1/sectors`
Sector indexes with nested constituent companies.

`data: [{ symbol, icon, type, last, change, percent, companies: [{ symbol, name, last, change, percent, volume, value, marketcap, icon_url, company_status, is_uma }] }]`

### `GET /v1/stocks`
IHSG constituent list. `data: [{ symbol, name, last, change, percent, volume, value, marketcap, icon_url, company_status, is_uma }]`

## Company fundamentals

### `GET /v1/company/{symbol}/profile`
Company profile. `symbol` is path param (required, uppercase as traded).

`data: { background, history, key_executive, address, subsidiary, beneficiary, shareholder, shareholder_director_commissioner, shareholder_numbers, shareholder_one_percent }`

### `GET /v1/company/{symbol}/subsidiaries`
Subsidiary list.

`data: { currency, last_updated_period, unit, subsidiaries: [{ company_name, business_type, location, commercial_year, total_assets, percentage, operational_status, period, raw }] }`

### `GET /v1/company/{symbol}/shareholding-composition`
Insider shareholding composition per reporting period.

| param | required | notes |
|---|---|---|
| `symbol` | path | |
| `period_start` | no | `YYYY-MM-DD`, filters upstream periods |
| `period_end` | no | `YYYY-MM-DD` |

`data: [{ report_date, total_shares: {raw, formatted}, compositions: [{ label, shares, percentage, colors }] }]`

### `GET /v1/insider/shareholding-network`
Shareholding network graph for a root node.

| param | required | values |
|---|---|---|
| `root_id` | yes | node id |
| `root_type` | yes | `SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR` (or `_COMPANY`) |
| `max_depth` | no | int (default upstream behavior) |
| `max_edge_per_node` | no | int |

`data: { root_id, root_type, report_date, nodes: [{id, node_type, metadata: {company|investor}, min_depth, is_rendered}], edges: [{from_id, to_id, shareholding, is_rendered}] }`

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

### `GET /v1/company/{symbol}/corp-actions`
Corporate action history. `action_info` is a typed payload dispatched by `action_type`.

| param | required | notes |
|---|---|---|
| `symbol` | path | |
| `limit` | no | int |

`data: [{ action_type, action_info: { rups | rightissue | stocksplit | <other>: {...} } }]`

### `GET /v1/company/{symbol}/keystats`
Key-stat ratios (display-formatted strings, ~10 years).

| param | required | notes |
|---|---|---|
| `symbol` | path | |
| `year_limit` | no | int |

`data: { closure_fin_items_results, financial_year_parent, stats, info, dividend_group, financial_report_currency }`

### `GET /v1/company/{symbol}/price-performance`
Price performance per timeframe. `data: { prices: [{ close, high, low, percentage, timeframe }] }`
(timeframes: `1D 1W 1M 3M 6M YTD 1Y`).

### `GET /v1/company/{symbol}/fundachart`
Raw historical ratio series for one or more fin-items. **This is the raw-number source
for keystats items.**

| param | required | values |
|---|---|---|
| `symbol` | path | |
| `item` | yes | fin-item id(s), comma-separated e.g. `2661,2525,1562` |
| `timeframe` | no | `1y 3y 5y 10y` (default `10y`) |

`data: [{ company_id, company_name, ratios: [{ decimal_point, group_data, item_id, item_name, item_type, suffix, xaxis_id, yaxis_id, chart_data: [{ date, formated_date, value, ratio_value }] }] }]`

### `GET /v1/fundachart/metrics`
Catalog of available `item` ids (recursive 3-level tree) to use with `/fundachart`.

| param | required | values |
|---|---|---|
| `metric_name` | yes | `fundachart` |

`data: [{ fitem_id, fitem_name, show_chart_icon, child: [...] }]`

### `GET /v1/company/{symbol}/financial`
Structured financial statement (II-a). No HTML; `data_tables` is the parsed table.

| param | required | values |
|---|---|---|
| `symbol` | path | |
| `data_type` | no | int (upstream: `1` returns data) |
| `is_percentage` | no | `0` = nominal value, `1` = percentage |
| `page` | yes | int ≥ 1 |
| `report_type` | yes | `1`=Income Statement, `2`=Balance Sheet, `3`=Cash Flow |
| `statement_type` | yes | `1`=Quarterly, `2`=Annual, `3`=TTM, `4`=Interim YTD, `5..8`=Q1..Q4, `9`=QoQ Growth, `10`=Quarter YoY, `11`=YTD YoY, `12`=Annual YoY, `13`=3Y CAGR |

`data: { currency, default_currency, rounding_value, data_tables: { periods, max_show_level, accounts: [{ id, level, name, values, accounts: [...], is_total_exist, is_default_expanded, max_show_level }] } }`

`values[i]` aligns with `periods[i]`; `"-"` means no data. `rounding_value` is the
unit divisor (e.g. `1000000000` = billions).

## Notes

- `symbol` path params are validated as `required` (422 on empty). Query enums are
  validated with `oneof`; invalid values → `422 VALIDATION_ERROR`.
- Upstream rate-limits aggressively when looping; keep a delay between calls when
  fetching many symbols.
- The stockbit client sends mobile headers (`X-Platform: iOS`) by default; some
  mobile-only endpoints (e.g. `/financial`) return empty data without them.