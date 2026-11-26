package notify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
)

var (
	privateHandle = regexp.MustCompile(`^@([A-Za-z][A-Za-z0-9_]{0,31})$`)
	publicGroup   = regexp.MustCompile(`(?i)^(?:https?://)?t\.me/([a-z][a-z0-9_]{0,31})/?$`)
	linkNonce     = regexp.MustCompile(`^[0-9a-f]{32}$`)
)

type telegramChat struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Type     string `json:"type"`
}

type telegramUpdate struct {
	ID      int64 `json:"update_id"`
	Message struct {
		Text string       `json:"text"`
		Date int64        `json:"date"`
		Chat telegramChat `json:"chat"`
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
	} `json:"message"`
}

type telegramStart struct {
	chat    telegramChat
	expires time.Time
}

// Keep recent Start confirmations for other users when paging through the bot's update queue.
type telegramLinks struct {
	sync.Mutex
	starts map[string]telegramStart
}

func destinationError(message string) error {
	return apperrors.ValidationError{Field: "telegramChatId", Message: message}
}

func (n *TelegramNotifier) ResolveDestination(ctx context.Context, owner domain.ID, input, nonce string) (string, error) {
	if match := publicGroup.FindStringSubmatch(input); match != nil {
		var chat telegramChat
		if err := n.lookup(ctx, "getChat", map[string]string{"chat_id": "@" + match[1]}, &chat); err != nil || chat.Type != "supergroup" || chat.ID >= 0 {
			return "", destinationError("Could not find that public group. Check its link and add the Vigil bot to the group.")
		}
		return strconv.FormatInt(chat.ID, 10), nil
	}
	handle := privateHandle.FindStringSubmatch(input)
	if handle == nil {
		return "", destinationError("Enter a personal @username, a public t.me group link or a numeric chat ID. Private invite links require a numeric chat ID.")
	}
	if !linkNonce.MatchString(nonce) {
		return "", destinationError("Open the personal chat link in Alerts and tap Start, then save your username.")
	}
	hash := sha256.Sum256([]byte(string(owner) + ":" + nonce))
	code := hex.EncodeToString(hash[:16])
	return n.findStart(ctx, code, handle[1])
}

func (n *TelegramNotifier) findStart(ctx context.Context, code, username string) (string, error) {
	n.links.Lock()
	defer n.links.Unlock()
	now := time.Now()
	if n.links.starts == nil {
		n.links.starts = make(map[string]telegramStart)
	}
	for key, start := range n.links.starts {
		if !start.expires.After(now) {
			delete(n.links.starts, key)
		}
	}
	var offset int64
	for page := 0; page < 5; page++ {
		if start, ok := n.links.starts[code]; ok && strings.EqualFold(start.chat.Username, username) {
			return strconv.FormatInt(start.chat.ID, 10), nil
		}
		var updates []telegramUpdate
		if err := n.lookup(ctx, "getUpdates", map[string]int64{"offset": offset, "limit": 100}, &updates); err != nil {
			return "", destinationError("Vigil could not check your Start message. Try again or use your numeric chat ID.")
		}
		for _, update := range updates {
			offset = update.ID + 1
			msg := update.Message
			key, ok := strings.CutPrefix(msg.Text, "/start vigil_")
			expiry := time.Unix(msg.Date, 0).Add(15 * time.Minute)
			if ok && linkNonce.MatchString(key) && msg.Chat.Type == "private" && msg.From.ID == msg.Chat.ID && msg.Chat.ID > 0 && expiry.After(now) {
				n.links.starts[key] = telegramStart{msg.Chat, expiry}
			}
		}
		if len(updates) < 100 {
			break
		}
	}
	if start, ok := n.links.starts[code]; ok && strings.EqualFold(start.chat.Username, username) {
		return strconv.FormatInt(start.chat.ID, 10), nil
	}
	return "", destinationError("Open the personal chat link below, tap Start, then save your own @username within 15 minutes.")
}

// Only call Telegram's fixed API endpoint. Transport errors can contain the bot token.
func (n *TelegramNotifier) lookup(ctx context.Context, method string, params any, result any) error {
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.endpoint+"/bot"+n.botToken+"/"+method, bytes.NewReader(body))
	if err != nil {
		return errors.New("telegram lookup unavailable")
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := n.client.Do(req)
	if err != nil {
		return errors.New("telegram lookup unavailable")
	}
	defer response.Body.Close()
	envelope := struct {
		OK     bool `json:"ok"`
		Result any  `json:"result"`
	}{Result: result}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(&envelope) != nil || !envelope.OK {
		return errors.New("telegram lookup unavailable")
	}
	return nil
}
