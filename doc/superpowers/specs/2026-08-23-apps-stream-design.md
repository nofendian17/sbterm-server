# Design: apps/stream — WebSocket fan-out untuk running trade

Tanggal: 2026-08-23
Status: disetujui (brainstorming session)

## Tujuan

Menyediakan endpoint WebSocket yang men-streaming data running trade ke klien eksternal. Sumber datanya adalah topik Kafka `datafeed.running_trade_batch` — payload protobuf yang diproduksi oleh `apps/ws` dari datafeed Stockbit. Aplikasi baru `apps/stream` menjadi consumer sekaligus hub fan-out; ia tidak menyentuh QuestDB maupun Stockbit langsung.

## Keputusan desain

| Pertanyaan | Keputusan |
|---|---|
| Cara client memilih data | Broadcast semua batch secara default; client dapat mengirim subscribe/unsubscribe untuk memfilter simbol tertentu |
| Lokasi layanan | App baru `apps/stream` (modul baru di workspace, total menjadi tujuh) |
| Format pesan ke client | JSON |
| Konsumsi Kafka | Consumer group sendiri (`sbterm-stream`), offset awal latest |
| Skema protokol | Subscribe/unsubscribe via JSON; setiap payload data dibungkus envelope bertipe |

## Struktur modul

Mengikuti pola `apps/ws` / `apps/ingest`:

```text
apps/stream/
  cmd/stream/main.go            # thin entry → container.Run()
  internal/
    container/container.go      # wiring + lifecycle
    service/
      hub.go                    # registry klien + fan-out terfilter
      client.go                 # buffer kirim per-koneksi + write pump
      ingest.go                 # poll loop Kafka → hub
    delivery/ws/
      server.go                 # chi + gorilla upgrader, GET /ws dan GET /healthz
      handler.go                # read pump: parse subscribe/unsubscribe
      message.go                # DTO protokol (envelope bertipe)
    infrastructure/
      config/config.go          # viper, config.stream.yaml
      kafka/consumer.go         # franz-go, consumer group, start AtEnd
```

Perubahan di luar app: tambah modul di `go.work`; target Makefile `run-stream`, serta iterasi `build`/`test`/`vet`/`tidy` mencakup `apps/stream`; service `stream` di `docker-compose.yml`; berkas `config.stream.yaml.example`.

## Aliran data

```text
Kafka (datafeed.running_trade_batch, protobuf)
  → poll loop [service/ingest]      decode RunningTradeBatch; error decode → log warn, lanjut record berikutnya
  → Hub.Broadcast                   iterasi klien terdaftar
      klien tanpa subscribe  → terima semua batch
      klien ber-subscribe    → cocokkan simbol batch (set lookup O(1))
  → per-klien channel buffer (256)  non-blocking send; penuh → disconnect klien lambat
  → write pump goroutine            marshal JSON sekali per batch, tulis frame teks
```

Unit fan-out adalah **batch** (satu simbol, banyak trade), sesuai bentuk payload Kafka — bukan per trade tunggal. Envelope data:

```json
{"type":"trade","symbol":"BBCA","data":[…trades…]}
```

JSON di-marshal **sekali per batch** dan hasilnya dibagikan ke semua klien yang cocok; write pump hanya menulis bytes yang sudah jadi.

## Protokol WebSocket

Client → server (teks/JSON):

```json
{"action":"subscribe","symbols":["BBCA","BBRI"]}
{"action":"unsubscribe","symbols":["BBCA"]}
```

- `subscribe` dengan `symbols: []` mengembalikan klien ke mode broadcast.
- `unsubscribe` dengan `symbols: []` tidak melakukan apa-apa (bukan disconnect).
- Pesan tidak valid atau action tak dikenal → balasan error, koneksi tetap hidup:
  `{"type":"error","message":"…"}`

Server → client:

```json
{"type":"trade","symbol":"BBCA","data":[…]}
{"type":"error","message":"…"}
```

Tidak ada pesan hello/auth (YAGNI). Keepalive standar gorilla: read deadline dengan pong handler, ping ticker ±45 detik.

## Kafka consumer

franz-go, pola sama dengan `apps/ingest`:

- Brokers, topic (`running_trade_batch_topic`, default `datafeed.running_trade_batch`), dan group (`kafka.group`, default `sbterm-stream`) dari config.
- Offset awal `AtEnd()` — konsumsi mulai dari data terbaru; tidak ada backfill riwayat.
- **Tanpa commit offset**: jalur fan-out murni, durabilitas adalah tanggung jawab `apps/ingest`.
- Fetch error → log warn dan tetap polling (jangan matikan loop).

## Lifecycle

Startup: `config.Load()` → logger → container (consumer, hub, server) → `Start()`. Tidak ada dependensi infrastruktur lain selain Kafka; koneksi Kafka dikelola franz-go (reconnect otomatis).

Shutdown (SIGTERM/SIGINT), berurutan dengan timeout total ≤5 detik:

1. Batalkan context poll loop (berhenti broadcast lebih dulu).
2. Tutup HTTP server (graceful).
3. Hub menutup semua sisa klien.

Klien yang menutup koneksi sendiri di-unregister otomatis; unregister aman terhadap double-call.

## Konfigurasi (`config.stream.yaml.example`)

```yaml
port: ":8081"   # port HTTP untuk /ws dan /healthz

kafka:
  brokers: ["redpanda:29092"]
  group: sbterm-stream
  running_trade_batch_topic: datafeed.running_trade_batch

log:
  level: info
  format: text
  add_source: false
```

## Endpoint HTTP

- `GET /ws` — upgrade WebSocket.
- `GET /healthz` — `{"status":"ok"}` untuk healthcheck compose/orkestrator; tidak mengecek Kafka karena reconnect ditangani franz-go sendiri.

## Testing

- `service/hub`: table-driven — filter per simbol, broadcast penuh, slow-client terputus saat buffer penuh, double-unregister aman; race detector.
- `delivery/ws`: `httptest` + dialer gorilla — subscribe lalu hanya menerima simbol terkait; malformed message → envelope error, koneksi hidup; ping/pong keepalive.
- `service/ingest`: fake consumer lewat interface (pola `committer` di ingest) — batch diteruskan ke hub; record rusak dilog dan dilewati.
- `infrastructure/kafka`: wiring/config test ringan (pola producer/consumer test yang ada).
- `container`: smoke test wiring; `config`: parsing default dan override file.

## Di luar lingkup (YAGNI)

Autentikasi/otorisasi klien, replay/backfill riwayat, kompresi pesan, metrik Prometheus, sharding multi-consumer, format protobuf ke klien.
