package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// 模拟并发数
const Concurrency = 20

// 总请求时间
const Duration = 30 * time.Second

func main() {
	url := "http://localhost:8888/v1/ranking/top?n=10"
	fmt.Println("开始压力测试 Ranking 服务...")
	fmt.Printf("并发数: %d, 持续时间: %v\n", Concurrency, Duration)
	fmt.Printf("目标 URL: %s\n", url)
	fmt.Println("--------------------------------------------------")

	var wg sync.WaitGroup
	start := time.Now()
	timeout := time.After(Duration)

	// 统计总请求数
	var count int64
	var mu sync.Mutex

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for i := 0; i < Concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-timeout:
					return
				default:
					resp, err := client.Get(url)
					if err == nil {
						// 读取完整 Body 确保连接复用
						resp.Body.Close()
						if resp.StatusCode == 200 {
							mu.Lock()
							count++
							mu.Unlock()
						}
					}
					// 适当休眠，避免本机过载
					// time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)
	qps := float64(count) / elapsed.Seconds()

	fmt.Println("--------------------------------------------------")
	fmt.Println("压测结束！")
	fmt.Printf("总成功请求数: %d\n", count)
	fmt.Printf("总耗时: %v\n", elapsed)
	fmt.Printf("平均 QPS: %.2f req/s\n", qps)
	fmt.Println("--------------------------------------------------")
	fmt.Println("现在去 Grafana 查看 Ranking 服务的监控图表吧！(QPS 应该会有明显飙升)")
}
