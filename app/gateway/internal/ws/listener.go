package ws

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	ChannelUserNotify = "ws:user:notify" // 定向推送: {"target_id":1, "payload":{...}}
	ChannelBroadcast  = "ws:broadcast"   // 全员广播: 任意 JSON
)

// StartRedisListener 订阅 Redis Pub/Sub，将消息路由给在线用户
// 在 Gateway main() 中以 goroutine 启动
func StartRedisListener(ctx context.Context, rdb *redis.Client) {
	pubsub := rdb.Subscribe(ctx, ChannelUserNotify, ChannelBroadcast)
	defer pubsub.Close()

	logx.Info("[WS] Redis listener started")

	for msg := range pubsub.Channel() {
		switch msg.Channel {
		case ChannelUserNotify:
			var envelope struct {
				TargetId int64           `json:"target_id"`
				Payload  json.RawMessage `json:"payload"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
				logx.Errorf("[WS] malformed notify: %v", err)
				continue
			}
			Hub.SendToUser(envelope.TargetId, json.RawMessage(envelope.Payload))

		case ChannelBroadcast:
			Hub.Broadcast(json.RawMessage(msg.Payload))
		}
	}
}
