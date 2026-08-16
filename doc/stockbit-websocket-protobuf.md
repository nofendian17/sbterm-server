# Stockbit Websocket Protobuf — Reverse Engineering Notes

Dokumen ini mencatat hasil riset implementasi protobuf di **https://stockbit.com/stream**
(analisis bundle webpack + capture wire langsung, 16 Agustus 2026).

> **Ringkasan**: halaman `/stream` menggunakan **REST JSON** untuk feed
> (exodus.stockbit.com), sedangkan **WebSocket membawa binary protobuf**
> (library **protobuf-es** / `@bufbuild/protobuf`). Semua definisi message
> di-reverse dari bundle frontend, bukan skema resmi Stockbit.

---

## 1. Arsitektur data

| Path | Format | Endpoint |
|---|---|---|
| Feed `/stream` | REST JSON | `https://exodus.stockbit.com/stream/v3?category=STREAM_CATEGORY_ALL_WATCHLIST&last_stream_id=0&limit=20` |
| Event live stream | REST JSON | `https://exodus.stockbit.com/live-stream/event?page=1&limit=20` |
| Reaksi stream | REST JSON | `https://exodus.stockbit.com/stream/reactions` |
| Realtime (WS) | **binary protobuf** | lihat §2 |

## 2. Endpoint WebSocket (3)

| Koneksi | URL | Isi |
|---|---|---|
| Social | `wss://wssocial.stockbit.com/?wskey=<key>` | chat (typing indicator, message) |
| Trading | `wss://wss-trading.stockbit.com/ws` | portfolio live, order, **nego engine order book** |
| Datafeed | `wss://wssfeed.stockbit.com/?wskey=<key>` | liveprice, entitas datafeed/financial, frame subscribe — tercatat di header semua proto datafeed & financial |
| Generic | `wss://ws-gen.stockbit.com/v1` | frame envelope umum (ping/auth/securities) |

Semua dijalankan lewat framework **Primus**:

```js
window.Primus("https://ws3.stockbit.com/", { strategy: false });
```

Handshake info: `https://ws3.stockbit.com/primus/info` →
`{"websocket":true,"origins":["*:*"],"cookie_needed":false,"entropy":736912565}`

> **Catatan**: info Primus berasal dari fase riset awal bundle; frame yang
> ter-capture adalah protobuf polos tanpa lapisan framing Primus. Koneksi
> final (tabel §2) memakai WebSocket langsung, bukan Primus.

## 3. Flow autentikasi (wskey)

1. `GET https://exodus.stockbit.com/auth/websocket/key`
   (Authorization: Bearer `<access token>`)
   → `{"message":"...","data":{"key":"..."}}` (sudah ada `GetWebsocketKey` di
   `internal/infrastructure/stockbit/websocket.go`)
2. wskey dipakai sebagai query param `?wskey=<key>` saat connect WS.
3. wskey juga dikirim dalam frame subscribe (datafeed `WebsocketRequest`
   field 3 = `key`; fungsi
   `convertWSKeyArray` di bundle mengubah array key menjadi byte array).
4. Cek otorisasi: `fetch(wsUrl.replace("wss://","https://"), {headers:{Authorization:"Bearer "+token}})`
   → 401 memicu re-login.

## 4. Protokol wire

Semua frame adalah **binary protobuf** standar (varint wire format).

### Client → Server

```js
// envelope — platform.websocket.wsevent.v1.WebsocketRequest (wsevent.proto)
new cA.vC({ requests: [request] }).toBinary()
// requests[] → Request{ value = 1 (bytes: session id / nilai channel),
//                       oneof { command = 2, subscribe = 3, unsubscribe = 4 } }
```

Frame subscribe nyata (941 byte, ter-capture):

```
field 1: session id        (bytes,  mis. "667557")
field 2: array channel     (mis. ["IHSG","TN.IHSG","NG.IHSG","SLIS",...])
field 3: wskey
```

Frame ini terbentuk dari **`securities.transactional.datafeed.v1.WebsocketRequest`**
(lihat §5): field 1 = `user_id`, field 2 = `channel.watchlist`, field 3 = `key`.

> **Dua level berbeda**: frame 941-byte di atas adalah level **datafeed**
> (`securities.transactional.datafeed.v1.WebsocketRequest` — field 1/2/3 =
> user_id/channel/key), sedangkan `cA.vC` adalah envelope level **platform**
> (`platform.websocket.wsevent.v1.WebsocketRequest` — field 1 = `requests[]`,
> tiap `Request` berisi `value`/`command`/`subscribe`/`unsubscribe`).
> Jangan tertukar saat implementasi.
Byte verbatim frame asli tidak direkam; wire-compat test
(`internal/infrastructure/stockbit/proto/wire_compat_test.go`) merekonstruksi
strukturnya (session + 108 channel + wskey placeholder = 744 byte; selisih
197 byte dari 941 adalah panjang wskey asli yang berupa token base64).
Hasil rekonstruksi round-trip deterministik: session dan channel ter-decode
kembali tanpa kehilangan data.

Ping: `IgYKBHBpbmc=` = `22 06 0A 04 70 69 6E 67`
= field 4 `ping` (length-delimited) → nested field 1 string `"ping"`
(datafeed `WebsocketRequest.ping`).
Subscribe di-debounce 100 ms (`cz()(_,100)`).

Kedua frame ping (`IgYKBHBpbmc=`, `CAE=`) telah diverifikasi **byte-exact**
dengan stub Go hasil protoc di `wire_compat_test.go` (Marshal Go ≡ frame
capture; decode frame asli → struktur yang diharapkan).

### Server → Client

```js
cA.XC.fromBinary(new Uint8Array(a.data))
// cA.XC = platform.websocket.wsevent.v1.WebsocketResponse
//  .response → oneof: ping | auth | securities
//  .timestamp → google.protobuf.Timestamp
```

Frame ping balasan: `CAE=` = `08 01` = field 1 varint 1
(proto3 oneof set tanpa nilai = value default).

## 5. Skema (di-reverse dari bundle)

Definisi lengkap ada di `internal/infrastructure/stockbit/proto/`, dengan
struktur direktori mengikuti package (wajib untuk `protoc`). Stubs Go
(`*.pb.go`) sudah di-generate dengan `protoc --go_out`:

| File `.proto` | Package | Isi |
|---|---|---|
| `platform/websocket/wsevent/v1/wsevent.proto` | `platform.websocket.wsevent.v1` | envelope request/response (channel enum: trade, securities live, ds-live/ds-live-order, etc.) |
| `securities/trm/core/portfolio/wsevent/v1/portfolio.proto` | `securities.trm.core.portfolio.wsevent.v1` | payload portfolio/order (action upsert/delete, message nego engine order book dkk.) |
| `securities/transactional/negoengine/order_book/entity/v1/order_book.proto` | `securities.transactional.negoengine.order_book.entity.v1` | order book realtime |
| `securities/transactional/datafeed/consumer/entity/v1/consumer_entity_v1.proto` | `securities.transactional.datafeed.consumer.entity.v1` | entitas datafeed (best_bid_offer, running_trade, ds_order_book dkk.) |
| `securities/transactional/datafeed/consumer/entity/v3/consumer_entity_v3.proto` | `securities.transactional.datafeed.consumer.entity.v3` | entitas datafeed iterasi v3 (liveprice, ds_order_book, miss/hit dkk.) |
| `securities/transactional/datafeed/v1/datafeed_websocket.proto` | `securities.transactional.datafeed.v1` | **WebsocketRequest datafeed** (field 1 user_id / 2 channel / 3 key / 4 ping → frame subscribe & ping trading) |
| `securities/transactional/negoengine/order/entity/v1/nego_engine_order.proto` | `securities.transactional.negoengine.order.entity.v1` | entitas order nego engine (pending order, ds_order) |
| `financial/common/v1/financial_common_v1.proto` | `financial.common.v1` | common financial (watchlist ticker) |
| `financial/company_price_feed/entity/v1/company_price_feed_entity_v1.proto` | `financial.company_price_feed.entity.v1` | entitas price feed (v1) |
| `financial/company_price_feed/entity/v2/company_price_feed_entity_v2.proto` | `financial.company_price_feed.entity.v2` | entitas price feed (v2: IEPIEV dkk.) |
| `financial/emitten/entity/v1/emitten_entity_v1.proto` | `financial.emitten.entity.v1` | entitas emitten (v1: movers dkk.) |
| `financial/emitten/entity/v2/emitten_entity_v2.proto` | `financial.emitten.entity.v2` | entitas emitten (v2) |
| `financial/order_trade/entity/v1/order_trade_entity_v1.proto` | `financial.order_trade.entity.v1` | entitas order/trade (watchlist_ticker dkk.) |
| `securities/trm/core/portfolio/entity/v2/portfolio_entity_v2.proto` | `securities.trm.core.portfolio.entity.v2` | entitas portfolio (v2) |
| `securities/trm/core/smart_order/entity/v1/smart_order.proto` | `securities.trm.core.smart_order.entity.v1` | entitas smart order |
| `securities/trm/core/smart_order/trailing_stop/order/entity/v1/trailing_stop.proto` | `securities.trm.core.smart_order.trailing_stop.order.entity.v1` | entitas trailing stop |
| `social/wssocial/entity/message/v1/social.proto` | `social.wssocial.entity.message.v1` | message sosial (stream/channel, ReceivedMessage type ping dkk.) |
| `social/wssocial/entity/wsevent/chat/v1/chat.proto` | `social.wssocial.entity.wsevent.chat.v1` | typing indicator, received message/room info |
| `social/wssocial/entity/wsevent/stream/v1/stream.proto` | `social.wssocial.entity.wsevent.stream.v1` | event stream social |
| `social/wssocial/entity/wsevent/research/v1/research.proto` | `social.wssocial.entity.wsevent.research.v1` | event research social |
| `social/stream/v1/stream_v1.proto` | `social.stream.v1` | stream entity (old response, GetStream dkk.) |
| `social/stream/entity/v1/stream_entity_v1.proto` | `social.stream.entity.v1` | entitas stream (user, reply_info, share_trade, following_activity) |
| `social/stream/data/v1/stream_data_v1.proto` | `social.stream.data.v1` | data stream (frame_type, commenter_type, spam_status) |
| `social/chat/entity/v2/chat_entity_v2.proto` | `social.chat.entity.v2` | entitas chat (room, message, attachment, share trade dkk.) |
| `social/verified_badge/enum/v1/verified_badge_enum_v1.proto` | `social.verified_badge.enum.v1` | enum verified status |
| `google/type/date.proto`, `google/type/decimal.proto`, `google/type/money.proto` | `google.type.*` | dependent google types di-vendor untuk kompilasi |

Sumber: chunk `99408.42434b7db9a1134e.js` webpack **module 32665** (definisi
message), chunk `27251-18bde4bd460a5138.js` (runtime protobuf-es),
chunk `51232-5d0c93927d38e370.js` **module 228** (datafeed/financial),
chunk `56256.ea110c7582406bb9.js` **module 45621** (wssocial:
chat/stream/research), chunk `56256...js` **module 1835** (wsevent chat v1).

> **Catatan**: skema seluruhnya hasil rekonstruksi defensif dari bundle —
> **bukan skema resmi** Stockbit. Header setiap `.proto` mencatat sumber
> chunk/module-nya. Stub Go yang dihasilkan kompatibel wire (lihat §4;
> diverifikasi di `wire_compat_test.go`).

### Regenerasi stubs Go

```sh
cd internal/infrastructure/stockbit/proto
protoc \
  -I . \
  --go_out=. --go_opt=module=github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto \
  $(find . -name '*.proto' | sort)
```

### Catatan pemetaan tipe (protobuf-es `ScalarType`)

`T:` pada field table = nomor tipe protobuf:

| T | Tipe |
|---|---|
| 4 | UINT64 |
| 5 | INT32 |
| 8 | BOOL |
| 9 | STRING |
| 12 | BYTES |

**Perhatian**: field table di bundle tidak selalu jujur — contoh
`social.Channel.roomlist` ditulis `kind:"scalar",T:8` tetapi nama asli
`"uint64"` di sumber adalah **bool** (T:8 = BOOL). Cek wire format (tag)
saat meragukan.

### Belum di-reverse

- Field table `PortfolioSummary`, `PortfolioDetail`, `PortfolioList`,
  `OrderDetail` (tipe direferensikan sebagai module terpisah di bundle,
  `h.XJ`/`h.$2`/`h.KM`).
- Format feed `wssocial` lengkap untuk channel selain chat/stream/research
  (Hanya ReceivedTypingIndicator, ReceivedMessage, ReceivedRoomInfo,
  ReceivedPost untuk stream & research yang dikenali).
- Field table beberapa payload `datafeed` consumer (v1/v3) yang masih berupa
  placeholder/referensi module.
- Enum `platform.websocket.wsevent.v1.WebsocketRequest.Request.Channel` masih
  **stub**: hanya `CHANNEL_SECURITIES_LIVE_PORTO` dan
  `CHANNEL_NEGO_ENGINE_ORDER_BOOK`; doc §5 menyebut "ds-live/ds-live-order,
  etc." — member tsb belum ada di skema. Hati-hati kalau dipakai untuk
  subscribe channel lain.

## 6. Jalur implementasi Go (WS client)

```
1. GET exodus.stockbit.com/auth/websocket/key   (sudah ada)
2. connect wss://wss-trading.stockbit.com/ws?wskey=<key> (atau tanpa query; lihat bundle)
   — datafeed: `wss://wssfeed.stockbit.com/?wskey=<key>` (header datafeed_websocket.proto; lihat §2)
3. kirim frame: WebsocketRequest{requests:[{value:session, subscribe:...}]}.toBinary()
4. terima frame → decode platform.websocket.wsevent.v1.WebsocketResponse
5. baca .response.securities
   → securities.trm.core.portfolio.wsevent.v1.WebsocketResponse
   → .message.nego_engine_order_book (field 8)
6. decode securities.transactional.negoengine.order_book.entity.v1.OrderBook
   → bid/ask entries (frequency, shares, price)
```

Library Go yang dipakai: `google.golang.org/protobuf` (sudah ada di go.mod
via generated stubs), WebSocket: `gorilla/websocket` / `nhooyr.io/websocket`.

## 7. Bukti wire (capture)

| Frame | Hex / Base64 | Arti |
|---|---|---|
| social ping balasan | `CAE=` → `08 01` | field 1 varint 1 |
| trading ping | `IgYKBHBpbmc=` → `22 06 0A 04 70 69 6E 67` | field 4 → "ping" |
| subscribe (ter-capture) | 941 byte: field 1 session, field 2 channel[], field 3 wskey | subscribe channel |
| subscribe (rekonstruksi) | 744 byte: `0A <len> session` + `12 <len> channel[]` + `1A <len> wskey` | `wire_compat_test.go` marshal stub Go → round-trip deterministik |

Semua perilaku di tabel diverifikasi `go test ./internal/infrastructure/stockbit/proto/ -run WireCompat`
(frame ping byte-exact terhadap capture; rekonstruksi subscribe round-trip).
File test: `internal/infrastructure/stockbit/proto/wire_compat_test.go`.

## 8. Referensi bundle (untuk riset lanjutan)

| Chunk | Isi |
|---|---|
| `99408.42434b7db9a1134e.js` | definisi message (module 32665) + enum |
| `27251-18bde4bd460a5138.js` | runtime protobuf-es (Reader/Writer, Timestamp) |
| `56256.ea110c7582406bb9.js` | integrasi Primus (connect, subscribe, ping) |
| `70906-39403424593c6acc.js` | redux-saga init websocket, convertWSKeyArray |
