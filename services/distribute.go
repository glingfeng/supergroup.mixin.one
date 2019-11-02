package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/supergroup.mixin.one/config"
	"github.com/MixinNetwork/supergroup.mixin.one/models"
	"github.com/MixinNetwork/supergroup.mixin.one/session"
)

var gshard map[string]bool

func distribute(ctx context.Context) {
	limit := int64(80)
	for i := int64(0); i < config.AppConfig.System.MessageShardSize; i++ {
		shard := shardId(config.AppConfig.System.MessageShardModifier, i)
		if i <= 5 {
			gshard[shard] = true
			log.Println("SHARD", gshard, i)
		}
		go pendingActiveDistributedMessages(ctx, shard, limit)
	}
}

func pendingActiveDistributedMessages(ctx context.Context, shard string, limit int64) {
	for {
		var t time.Time
		if gshard[shard] {
			t = time.Now()
		}
		_, err := models.CleanUpExpiredDistributedMessages(ctx, shard)
		if err != nil {
			session.Logger(ctx).Errorf("CleanUpExpiredDistributedMessages ERROR: %+v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if gshard[shard] {
			session.Logger(ctx).Infof("PendingActiveDistributedMessages CleanUpExpiredDistributedMessages %s TIME::: %v", shard, time.Now().Sub(t))
		}
		messages, err := models.PendingActiveDistributedMessages(ctx, shard, limit)
		if err != nil {
			session.Logger(ctx).Errorf("PendingActiveDistributedMessages ERROR: %+v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if len(messages) < 1 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		err = sendDistributedMessges(ctx, shard, messages)
		if err != nil {
			session.Logger(ctx).Errorf("PendingActiveDistributedMessages sendDistributedMessges ERROR: %+v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if gshard[shard] {
			session.Logger(ctx).Infof("PendingActiveDistributedMessages sendDistributedMessges %s TIME::: %v", shard, time.Now().Sub(t))
		}
		err = models.UpdateMessagesStatus(ctx, messages)
		if err != nil {
			session.Logger(ctx).Errorf("PendingActiveDistributedMessages UpdateMessagesStatus ERROR: %+v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if gshard[shard] {
			session.Logger(ctx).Infof("PendingActiveDistributedMessages UpdateMessagesStatus %s TIME::: %v", shard, time.Now().Sub(t))
		}
	}
}

func sendDistributedMessges(ctx context.Context, key string, messages []*models.DistributedMessage) error {
	var body []map[string]interface{}
	for _, message := range messages {
		if message.UserId == config.AppConfig.Mixin.ClientId {
			message.UserId = ""
		}
		if message.Category == models.MessageCategoryMessageRecall {
			message.UserId = ""
		}
		body = append(body, map[string]interface{}{
			"conversation_id":   message.ConversationId,
			"recipient_id":      message.RecipientId,
			"message_id":        message.MessageId,
			"quote_message_id":  message.QuoteMessageId,
			"category":          message.Category,
			"data":              message.Data,
			"representative_id": message.UserId,
			"created_at":        message.CreatedAt,
			"updated_at":        message.CreatedAt,
		})
	}

	msgs, err := json.Marshal(body)
	if err != nil {
		return err
	}
	mixin := config.AppConfig.Mixin
	accessToken, err := bot.SignAuthenticationToken(mixin.ClientId, mixin.SessionId, mixin.SessionKey, "POST", "/messages", string(msgs))
	if err != nil {
		return err
	}
	data, err := request(ctx, key, "POST", "/messages", msgs, accessToken)
	if err != nil {
		return err
	}
	var resp struct {
		Error bot.Error `json:"error"`
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}

var httpPool map[string]*http.Client = make(map[string]*http.Client, 0)

func request(ctx context.Context, key, method, path string, body []byte, accessToken string) ([]byte, error) {
	if httpPool[key] == nil {
		httpPool[key] = &http.Client{Timeout: 3 * time.Second}
	}
	req, err := http.NewRequest(method, "https://api.mixin.one"+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := httpPool[key].Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, bot.ServerError(ctx, nil)
	}
	return ioutil.ReadAll(resp.Body)
}
