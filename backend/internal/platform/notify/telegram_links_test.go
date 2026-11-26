package notify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResolveTelegramDestination(t *testing.T) {
	const nonce = "1234567890abcdef1234567890abcdef"
	hash := sha256.Sum256([]byte("owner:" + nonce))
	command := "/start vigil_" + hex.EncodeToString(hash[:16])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bottest/getUpdates":
			fmt.Fprintf(w, `{"ok":true,"result":[{"update_id":17,"message":{"text":%q,"date":%d,"chat":{"type":"private","username":"A_User","id":12345},"from":{"id":12345}}}]}`, command, time.Now().Unix())
		case "/bottest/getChat":
			var args map[string]string
			if err := json.NewDecoder(r.Body).Decode(&args); err != nil || args["chat_id"] != "@Group_1" {
				t.Errorf("bad group request: %v", args)
			}
			fmt.Fprint(w, `{"ok":true,"result":{"type":"supergroup","id":-100123456}}`)
		default:
			t.Errorf("unexpected method %s", r.URL.Path)
		}
	}))
	defer server.Close()
	n := NewTelegramNotifier("test")
	n.endpoint = server.URL
	for _, input := range []string{"https://t.me/Group_1", "t.me/Group_1/"} {
		if id, err := n.ResolveDestination(context.Background(), "owner", input, ""); err != nil || id != "-100123456" {
			t.Fatalf("group lookup: %q, %v", id, err)
		}
	}
	if id, err := n.ResolveDestination(context.Background(), "owner", "@a_user", nonce); err != nil || id != "12345" {
		t.Fatalf("private lookup: %q, %v", id, err)
	}
	for _, input := range []string{"@other", "t.me/+invite", "https://elsewhere.example/Group_1", "t.me/Group_1/123"} {
		if _, err := n.ResolveDestination(context.Background(), "owner", input, nonce); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
	if _, err := n.ResolveDestination(context.Background(), "other-owner", "@a_user", nonce); err == nil {
		t.Error("accepted another account's link")
	}
}

func TestRejectUnconfirmedTelegramStart(t *testing.T) {
	const nonce = "abcdef1234567890abcdef1234567890"
	hash := sha256.Sum256([]byte("owner:" + nonce))
	command := "/start vigil_" + hex.EncodeToString(hash[:16])
	for _, tc := range []struct {
		name, kind string
		date, from int64
	}{
		{"expired", "private", time.Now().Add(-16 * time.Minute).Unix(), 12345},
		{"group", "group", time.Now().Unix(), 12345},
		{"different sender", "private", time.Now().Unix(), 23456},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"ok":true,"result":[{"message":{"text":%q,"date":%d,"chat":{"type":%q,"username":"alice","id":12345},"from":{"id":%d}}}]}`, command, tc.date, tc.kind, tc.from)
			}))
			defer server.Close()
			n := NewTelegramNotifier("test")
			n.endpoint = server.URL
			if _, err := n.ResolveDestination(context.Background(), "owner", "@alice", nonce); err == nil {
				t.Fatal("accepted invalid confirmation")
			}
		})
	}
}
