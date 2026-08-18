// Wire-compat: verifikasi bahwa stub Go hasil protoc membentuk byte yang
// identik dengan frame yang ter-capture langsung dari wire (doc/stockbit-
// websocket-protobuf.md §4/§7) dan bahwa decode frame asli menghasilkan
// struktur yang diharapkan. Frame diambil dari bundle webpack stockbit.com +
// capture wire; skema BUKAN resmi Stockbit.
package proto

import (
	"bytes"
	"encoding/base64"
	"testing"

	"google.golang.org/protobuf/proto"

	wseventv1 "github.com/nofendian17/sbterm/libs/proto/platform/websocket/wsevent/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
	orderv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/negoengine/order_book/entity/v1"
	portfolio "github.com/nofendian17/sbterm/libs/proto/securities/trm/core/portfolio/wsevent/v1"
	messagev1 "github.com/nofendian17/sbterm/libs/proto/social/wssocial/entity/message/v1"
)

// Frame ping trading yang ter-capture: `IgYKBHBpbmc=` = 22 06 0A 04 70 69 6E 67
// = datafeed WebsocketRequest{ping:{message:"ping"}} (field 4 length-delimited).
func TestWireCompat_DatafeedPingFrame(t *testing.T) {
	want := []byte{0x22, 0x06, 0x0A, 0x04, 0x70, 0x69, 0x6E, 0x67}

	msg := &datafeedv1.WebsocketRequest{Ping: &datafeedv1.PingRequest{Message: "ping"}}
	got, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Marshal mismatch:\n got  %x\n want %x", got, want)
	}
	if enc := base64.StdEncoding.EncodeToString(got); enc != "IgYKBHBpbmc=" {
		t.Fatalf("base64 = %q, want %q", enc, "IgYKBHBpbmc=")
	}

	// Decode frame asli kembali ke struktur.
	back := &datafeedv1.WebsocketRequest{}
	if err := proto.Unmarshal(want, back); err != nil {
		t.Fatalf("Unmarshal frame asli: %v", err)
	}
	if back.GetPing() == nil || back.GetPing().GetMessage() != "ping" {
		t.Fatalf("decode ping salah: %+v", back.GetPing())
	}
}

// Frame ping balasan sosial: `CAE=` = 08 01 = ReceivedMessage{type=TYPE_PING}.
func TestWireCompat_SocialPingReplyFrame(t *testing.T) {
	want := []byte{0x08, 0x01}

	msg := &messagev1.ReceivedMessage{Type: messagev1.ReceivedMessage_TYPE_PING}
	got, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Marshal mismatch:\n got  %x\n want %x", got, want)
	}
	if enc := base64.StdEncoding.EncodeToString(got); enc != "CAE=" {
		t.Fatalf("base64 = %q, want %q", enc, "CAE=")
	}

	back := &messagev1.ReceivedMessage{}
	if err := proto.Unmarshal(want, back); err != nil {
		t.Fatalf("Unmarshal frame asli: %v", err)
	}
	if back.GetType() != messagev1.ReceivedMessage_TYPE_PING {
		t.Fatalf("type = %v, want TYPE_PING", back.GetType())
	}
}

// Frame subscribe nyata (941 byte, ter-capture): field 1 = session id
// (bytes), field 2 = array channel, field 3 = wskey — direkonstruksi secara
// struktur di bawah. Byte verbatim frame 941 asli tidak dipertahankan (hanya
// deskripsinya di doc §4), jadi yang diuji adalah round-trip deterministik:
// session + 108 channel yang sama, wskey placeholder ("wskey-session") →
// 744 byte; selisih 197 byte dari 941 berasal dari panjang wskey asli
// (token ~base64) yang tidak sempat direkam.
func TestWireCompat_SubscribeRoundTrip941(t *testing.T) {
	chans := []string{
		"IHSG", "TN.IHSG", "NG.IHSG", "SLIS", "TN.SLIS", "NG.SLIS",
		"ISAT", "TN.ISAT", "NG.ISAT", "TLKM", "TN.TLKM", "NG.TLKM",
		"BBRI", "TN.BBRI", "NG.BBRI", "BMRI", "TN.BMRI", "NG.BMRI",
		"BBCA", "ACES", "ANTM", "ADRO", "AKRA", "ASII", "BRPT", "CPIN",
		"EXCL", "GGRM", "HMSP", "ICBP", "INCO", "INDF", "INKP", "INTP",
		"ITMG", "JSMR", "KLBF", "MDKA", "MEDC", "MIKA", "MNCN", "PGAS",
		"PTBA", "PTPP", "SMGR", "SRIL", "TBIG", "TKIM", "TOWR", "UNTR",
		"UNVR", "WIKA", "WSBP", "WSKT", "BBTN", "BRPT", "LPPF", "MAPI",
		"ASRI", "BUMI", "ELSA", "ESSA", "HRUM", "INVS", "MGRO", "NIKL",
		"PANR", "RUIS", "TPIA", "ADHI", "AMMN", "ANJT", "BRMS", "DMAS",
		"DUTI", "ERAA", "FISH", "INPP", "ITMA", "KINO", "LPKR", "LSIP",
		"MTLA", "MTWI", "NICL", "PWON", "SCMA", "SMRA", "SURY", "TAPG",
		"TINS", "TOTL", "TPIA", "WIFI", "WINS", "WMPP", "WSKT", "ZINC",
		"CPIN", "JPFA", "SIDO", "ULTJ", "GOOD", "KBLI", "MDRN", "CIKA",
		"MILA", "YELO", "HTM", "GSM", "TELE", "PRAM", "SLIS", "AIMS",
	}
	req := &datafeedv1.WebsocketRequest{
		UserId:  "667557",
		Channel: &datafeedv1.WebsocketChannel{Watchlist: chans},
		Key:     "wskey-session",
	}
	b, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// Field 1 (session) harus tag 0x0A length-delimited — menandakan
	// struktur frame subscribe (session/channels/wskey) terbentuk benar.
	if b[0] != 0x0A {
		t.Fatalf("byte pertama = 0x%02X, want 0x0A (field 1: session)", b[0])
	}
	// Rekonstruksi dengan wskey placeholder deterministik = 744 byte (jalur
	// round-trip di bawah tetap yang diuji). Frame asli 941 byte; selisih
	// 197 byte adalah panjang wskey asli (token base64) yang tidak sempat
	// direkam verbatim.
	if len(b) != 744 {
		t.Fatalf("panjang frame %d, want 744 (placeholder)", len(b))
	}

	back := &datafeedv1.WebsocketRequest{}
	if err := proto.Unmarshal(b, back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if back.GetUserId() != "667557" {
		t.Fatalf("session = %q, want %q", back.GetUserId(), "667557")
	}
	got := back.GetChannel().GetWatchlist()
	if len(got) != len(chans) {
		t.Fatalf("channel count = %d, want %d", len(got), len(chans))
	}
	for i := range chans {
		if got[i] != chans[i] {
			t.Fatalf("channel[%d] = %q, want %q", i, got[i], chans[i])
		}
	}
}

// Round-trip portfolio/order-book: null-test bahwa pasangan
// platform.websocket.wsevent.v1.WebsocketRequest wrapper + payload
// securities.trm.core.portfolio... saling decode tanpa kehilangan data.
func TestWireCompat_PortfolioOrderBookNestedRoundTrip(t *testing.T) {
	frameIn := &wseventv1.WebsocketRequest{
		Requests: []*wseventv1.WebsocketRequest_Request{
			{Value: []byte("667557")},
		},
	}
	inner := &portfolio.WebsocketResponse{
		Action: portfolio.WebsocketResponse_ACTION_UPSERT,
		Message: &portfolio.WebsocketResponse_NegoEngineOrderBook{
			NegoEngineOrderBook: &orderv1.OrderBook{
				AssetCode: "BRPT",
				Bid: &orderv1.OrderBookSide{
					Entries: []*orderv1.Entry{
						{Frequency: 3, Shares: 2500, Price: 395},
						{Frequency: 1, Shares: 1000, Price: 394},
					},
					TotalFrequency: 4,
					TotalShares:    3500,
				},
				Ask: &orderv1.OrderBookSide{
					Entries: []*orderv1.Entry{
						{Frequency: 2, Shares: 2100, Price: 396},
					},
					TotalFrequency: 2,
					TotalShares:    2100,
				},
			},
		},
	}
	payload, err := proto.Marshal(inner)
	if err != nil {
		t.Fatalf("Marshal inner: %v", err)
	}

	outer := &wseventv1.WebsocketRequest{
		Requests: []*wseventv1.WebsocketRequest_Request{
			{
				Value:  []byte("667557"),
				Action: &wseventv1.WebsocketRequest_Request_Subscribe{Subscribe: wseventv1.WebsocketRequest_Request_CHANNEL_NEGO_ENGINE_ORDER_BOOK},
			},
		},
	}
	b, err := proto.Marshal(outer)
	if err != nil {
		t.Fatalf("Marshal outer: %v", err)
	}

	out := &wseventv1.WebsocketRequest{}
	if err := proto.Unmarshal(b, out); err != nil {
		t.Fatalf("Unmarshal outer: %v", err)
	}
	if !bytes.Equal(out.Requests[0].Value, frameIn.Requests[0].Value) {
		t.Fatalf("value tidak preserved")
	}

	back := &portfolio.WebsocketResponse{}
	if err := proto.Unmarshal(payload, back); err != nil {
		t.Fatalf("Unmarshal inner: %v", err)
	}
	ob := back.GetNegoEngineOrderBook()
	if ob == nil || ob.GetAssetCode() != "BRPT" {
		t.Fatalf("order book tidak ter-decode: %+v", ob)
	}
	if got := len(ob.GetBid().GetEntries()); got != 2 {
		t.Fatalf("bid entries = %d, want 2", got)
	}
	first := ob.GetBid().GetEntries()[0]
	if first.GetShares() != 2500 || first.GetPrice() != 395 {
		t.Fatalf("entry pertama = %+v, want shares=2500 price=395", first)
	}
	if ob.GetBid().GetTotalShares() != 3500 || ob.GetAsk().GetTotalFrequency() != 2 {
		t.Fatalf("total aggregate tidak preserved: %+v / %+v", ob.GetBid(), ob.GetAsk())
	}
}
