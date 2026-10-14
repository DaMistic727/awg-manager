package wdttlink

import (
	"strings"
	"testing"

	"github.com/hoaxisr/awg-manager/internal/storage"
)

const sampleEventLog = `
[КЛИЕНТ] Слушаю: 127.0.0.1:9000
__WDTT_EVENT__|CONFIG|{"config":"[Interface]\nPrivateKey = abc\nAddress = 10.0.0.2/32\n\n[Peer]\nPublicKey = xyz\nAllowedIPs = 0.0.0.0/0\n"}
`

const sampleBoxLog = `
╔══════════════ WireGuard Конфиг ══════════════╗
║ [Interface]                                  ║
║ PrivateKey = abc                             ║
║ Address = 10.0.0.2/32                        ║
║ [Peer]                                       ║
║ PublicKey = xyz                              ║
║ AllowedIPs = 0.0.0.0/0                       ║
╚══════════════════════════════════════════════╝
`

func TestExtractWGConfigFromLog_Event(t *testing.T) {
	got := ExtractWGConfigFromLog(sampleEventLog)
	if !looksLikeWGConf(got) {
		t.Fatalf("got=%q", got)
	}
	if !strings.Contains(got, "PrivateKey = abc") {
		t.Fatalf("missing private key: %q", got)
	}
}

func TestExtractWGConfigFromLog_Box(t *testing.T) {
	got := ExtractWGConfigFromLog(sampleBoxLog)
	if !looksLikeWGConf(got) {
		t.Fatalf("got=%q", got)
	}
}
func TestExtractPeerPublicKey(t *testing.T) {
	conf := "[Interface]\nPrivateKey = x\n\n[Peer]\nPublicKey = abc123=\n"
	if got := ExtractPeerPublicKey(conf); got != "abc123=" {
		t.Fatalf("got=%q", got)
	}
}

func TestPeersEqual(t *testing.T) {
	if !PeersEqual("1.2.3.4", "1.2.3.4:56000") {
		t.Fatal("expected equal with default port")
	}
	if PeersEqual("1.2.3.4:56000", "5.6.7.8:56000") {
		t.Fatal("expected different peers")
	}
}

func TestMatchingAWGTunnel(t *testing.T) {
	conf := "[Interface]\nPrivateKey = x\nAddress = 10.8.0.5/32\n\n[Peer]\nPublicKey = abc123=\n"

	// Same interface address but a different peer key must NOT match
	// (не усыновляем чужой туннель по совпавшему 10.x-адресу).
	tunnels := []storage.AWGTunnel{{
		ID:        "t1",
		Name:      "Германия wdtt",
		Peer:      storage.AWGPeer{PublicKey: "other="},
		Interface: storage.AWGInterface{Address: "10.8.0.5/32"},
	}}
	if got := MatchingAWGTunnel(tunnels, conf); got != nil {
		t.Fatalf("expected nil for addr-only overlap, got %v", got)
	}

	// Matching peer key (different address) → match.
	tunnels[0].Interface.Address = "10.9.0.1/32"
	tunnels[0].Peer.PublicKey = "abc123="
	if got := MatchingAWGTunnel(tunnels, conf); got == nil || got.ID != "t1" {
		t.Fatalf("expected match by pubkey, got %v", got)
	}

	// Empty config → no match, no panic.
	if got := MatchingAWGTunnel(tunnels, ""); got != nil {
		t.Fatalf("expected nil for empty conf, got %v", got)
	}
}

func TestPatchWgConfEndpoint(t *testing.T) {
	in := "[Interface]\nPrivateKey = x\n\n[Peer]\nPublicKey = y\nAllowedIPs = 0.0.0.0/0\n"
	got := PatchWgConfEndpoint(in, 9000)
	if !strings.Contains(got, "Endpoint = 127.0.0.1:9000") {
		t.Fatalf("got=%q", got)
	}
}

func TestExtractInterfaceAddress(t *testing.T) {
	conf := "[Interface]\nPrivateKey = p\nAddress = 10.66.0.4/32, fd00::4/128\nDNS = 10.66.0.1\n\n[Peer]\nAddress = 9.9.9.9/32\n"
	if got := ExtractInterfaceAddress(conf); got != "10.66.0.4/32, fd00::4/128" {
		t.Fatalf("Address = %q", got)
	}
	if got := ExtractInterfaceAddress("[Peer]\nPublicKey = k\n"); got != "" {
		t.Fatalf("без [Interface] ожидали пусто, got %q", got)
	}
}

// F390: «порт есть» — по разбору адреса, а не по двоеточию. Голый IPv6
// уходил в ссылку без порта; литерал берётся в скобки. След старой ошибки
// (`v6:порт` без скобок) не трогается — его разбирает splitPeerHostPort.
func TestNormalizePeerIPv6(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"2001:db8::1", "[2001:db8::1]:56000"},
		{"[2001:db8::1]", "[2001:db8::1]:56000"},
		{"[2001:db8::1]:443", "[2001:db8::1]:443"},
		{"1.2.3.4", "1.2.3.4:56000"},
		{"vpn.example.org:1", "vpn.example.org:1"},
		{"2001:db8::1:56000x", "2001:db8::1:56000x"},
	} {
		if got := normalizePeer(tc.in); got != tc.want {
			t.Errorf("normalizePeer(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	host, port, err := splitPeerHostPort(normalizePeer("2001:db8::1"))
	if err != nil || host != "2001:db8::1" || port != 56000 {
		t.Errorf("split = (%q, %d, %v), want (2001:db8::1, 56000)", host, port, err)
	}
}

// F661: IPv6-хост двоеточная форма не переносит — ссылка собирается в
// JSON-форме и разбирается нашим же импортом обратно в тот же peer; IPv4
// остаётся двоеточным.
func TestEncodeLinkIPv6RoundTrip(t *testing.T) {
	link, err := EncodeLinkWithClientPort("[2001:db8::1]:56004", 56001, "secret", []string{"h1"}, "Дом", 9000)
	if err != nil {
		t.Fatal(err)
	}
	p, err := parseWdttURI(link)
	if err != nil {
		t.Fatalf("разбор %q: %v", link, err)
	}
	if p.Peer != "[2001:db8::1]:56004" || p.Password != "secret" || p.Name != "Дом" ||
		p.Listen != "127.0.0.1:9000" || p.Workers != 16 || len(p.VKHashes) != 1 || p.VKHashes[0] != "h1" {
		t.Fatalf("payload = %+v", p)
	}
	v4, _ := EncodeLinkWithClientPort("1.2.3.4:56004", 56001, "secret", nil, "", 9000)
	if v4 != "wdtt://1.2.3.4:56004:56001:9000:secret" {
		t.Fatalf("IPv4-ссылка изменилась: %q", v4)
	}
}
