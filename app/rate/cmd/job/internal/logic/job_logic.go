package logic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"freeexchanged/app/rate/cmd/job/internal/config"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	// 免费汇率 API，无需注册，每小时更新一次
	apiURL         = "https://open.er-api.com/v6/latest/%s"
	rateKeyPrefix  = "rate:"
	updatedAtField = "_updated_at"
	rateKeyTTL     = 3600 // 1小时 TTL，Job 挂了数据还能撑 1 小时
)

// 支持的基准货币（每种都要拉一次 API）
var baseCurrencies = []string{
	"USD", "CNY", "EUR", "GBP", "JPY",
	"HKD", "KRW", "SGD", "AUD", "CAD",
}

// API 响应结构
type apiResponse struct {
	Result   string             `json:"result"`
	BaseCode string             `json:"base_code"`
	Rates    map[string]float64 `json:"rates"`
}

// Run 启动定时拉取任务
func Run(c config.Config) {
	rds := redis.MustNewRedis(c.BizRedis)
	client := &http.Client{Timeout: 10 * time.Second}

	logx.Info("[RateJob] started, fetching rates every 1 minute...")

	// 启动时立刻执行一次，不等第一个 tick
	fetchAll(rds, client)

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		fetchAll(rds, client)
	}
}

// fetchAll 拉取所有基准货币的汇率
func fetchAll(rds *redis.Redis, client *http.Client) {
	for _, base := range baseCurrencies {
		if err := fetchAndCache(rds, client, base); err != nil {
			// 单个货币失败不影响其他货币，记录日志继续
			logx.Errorf("[RateJob] failed to fetch %s: %v", base, err)
			continue
		}
		logx.Infof("[RateJob] successfully updated rates for %s", base)
	}
}

// fetchAndCache 拉取单个基准货币的汇率并写入 Redis
func fetchAndCache(rds *redis.Redis, client *http.Client, base string) error {
	// 1. 调用外部 API
	url := fmt.Sprintf(apiURL, base)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("http get failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body failed: %w", err)
	}

	// 2. 解析响应
	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}
	if apiResp.Result != "success" {
		return fmt.Errorf("api returned non-success result: %s", apiResp.Result)
	}

	// 3. 构建 field-value map 写入 Redis Hash
	// HSET rate:USD CNY 7.253400 EUR 0.923400 ... _updated_at 1708300000
	redisKey := rateKeyPrefix + base
	now := strconv.FormatInt(time.Now().Unix(), 10)

	kvMap := make(map[string]string, len(apiResp.Rates)+1)
	for currency, rateVal := range apiResp.Rates {
		kvMap[currency] = strconv.FormatFloat(rateVal, 'f', 6, 64)
	}
	kvMap[updatedAtField] = now // 记录最后更新时间

	if err := rds.Hmset(redisKey, kvMap); err != nil {
		return fmt.Errorf("redis hmset failed: %w", err)
	}

	// 4. 设置 TTL（1小时兜底）
	if err := rds.Expire(redisKey, rateKeyTTL); err != nil {
		// TTL 设置失败不是致命错误，记录日志即可
		logx.Errorf("[RateJob] set ttl failed for %s: %v", redisKey, err)
	}

	return nil
}
