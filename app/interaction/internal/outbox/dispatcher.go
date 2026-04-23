package outbox

import (
	"context"
	"fmt"
	"os"
	"time"

	"freeexchanged/pkg/eventstream"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const (
	statusPending    int64 = 0
	statusSent       int64 = 1
	statusProcessing int64 = 2
)

type Config struct {
	DataSource          string
	PollIntervalSeconds int `json:",default=2"`
	BatchSize           int `json:",default=100"`
	ClaimTimeoutSeconds int `json:",default=30"`
	Kafka               struct {
		Brokers []string
	}
}

type Dispatcher struct {
	conn      sqlx.SqlConn
	brokers   []string
	batchSize int
	interval  time.Duration
	claimTTL  time.Duration
	workerID  string
	producers map[string]*eventstream.Producer
}

type Event struct {
	ID       int64  `db:"id"`
	Topic    string `db:"topic"`
	EventKey string `db:"event_key"`
	Payload  string `db:"payload"`
}

func NewDispatcher(c Config) *Dispatcher {
	batchSize := c.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	intervalSeconds := c.PollIntervalSeconds
	if intervalSeconds <= 0 {
		intervalSeconds = 2
	}

	claimTimeoutSeconds := c.ClaimTimeoutSeconds
	if claimTimeoutSeconds <= 0 {
		claimTimeoutSeconds = 30
	}

	return &Dispatcher{
		conn:      sqlx.NewMysql(c.DataSource),
		brokers:   c.Kafka.Brokers,
		batchSize: batchSize,
		interval:  time.Duration(intervalSeconds) * time.Second,
		claimTTL:  time.Duration(claimTimeoutSeconds) * time.Second,
		workerID:  newWorkerID(),
		producers: make(map[string]*eventstream.Producer),
	}
}

func (d *Dispatcher) Run(ctx context.Context) {
	logx.Info("interaction outbox dispatcher started")
	defer d.Close()

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	d.dispatchOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.dispatchOnce(ctx)
		}
	}
}

func (d *Dispatcher) Close() {
	for _, producer := range d.producers {
		_ = producer.Close()
	}
}

func (d *Dispatcher) dispatchOnce(ctx context.Context) {
	pending, err := d.claimPending(ctx)
	if err != nil {
		logx.Errorf("load interaction outbox events failed: %v", err)
		return
	}

	for _, event := range pending {
		if ctx.Err() != nil {
			return
		}

		if err := d.publish(ctx, event); err != nil {
			logx.Errorf("publish interaction outbox event id=%d topic=%s failed: %v", event.ID, event.Topic, err)
			if markErr := d.markFailed(ctx, event.ID, err); markErr != nil {
				logx.Errorf("mark interaction outbox event id=%d failed: %v", event.ID, markErr)
			}
			continue
		}

		if err := d.markSent(ctx, event.ID); err != nil {
			logx.Errorf("mark interaction outbox event id=%d sent failed: %v", event.ID, err)
		}
	}
}

func (d *Dispatcher) claimPending(ctx context.Context) ([]Event, error) {
	if _, err := d.conn.ExecCtx(ctx, `
UPDATE interaction_outbox_events
SET status = ?, locked_by = ?, locked_until = ?, update_time = NOW()
WHERE
  (status = ? AND next_retry_at <= NOW())
  OR (status = ? AND locked_until < NOW())
ORDER BY id
LIMIT ?`, statusProcessing, d.workerID, time.Now().Add(d.claimTTL), statusPending, statusProcessing, d.batchSize); err != nil {
		return nil, err
	}

	var pending []Event
	err := d.conn.QueryRowsCtx(ctx, &pending, `
SELECT id, topic, event_key, CAST(payload AS CHAR) AS payload
FROM interaction_outbox_events
WHERE status = ? AND locked_by = ?
ORDER BY id
LIMIT ?`, statusProcessing, d.workerID, d.batchSize)
	return pending, err
}

func (d *Dispatcher) publish(ctx context.Context, event Event) error {
	producer, ok := d.producers[event.Topic]
	if !ok {
		producer = eventstream.NewProducer(eventstream.KafkaConf{Brokers: d.brokers}, event.Topic)
		d.producers[event.Topic] = producer
	}
	return producer.Publish(ctx, event.EventKey, []byte(event.Payload))
}

func (d *Dispatcher) markSent(ctx context.Context, id int64) error {
	_, err := d.conn.ExecCtx(ctx, `
UPDATE interaction_outbox_events
SET status = ?, locked_by = '', locked_until = NULL, last_error = '', update_time = NOW()
WHERE id = ? AND locked_by = ?`, statusSent, id, d.workerID)
	return err
}

func (d *Dispatcher) markFailed(ctx context.Context, id int64, publishErr error) error {
	retryCount, err := d.retryCount(ctx, id)
	if err != nil {
		return err
	}

	delaySeconds := 60
	if retryCount < 6 {
		delaySeconds = 1 << retryCount
	}

	message := publishErr.Error()
	if len(message) > 1024 {
		message = message[:1024]
	}

	_, err = d.conn.ExecCtx(ctx, `
UPDATE interaction_outbox_events
SET status = ?, retry_count = retry_count + 1, last_error = ?, next_retry_at = ?, locked_by = '', locked_until = NULL, update_time = NOW()
WHERE id = ? AND locked_by = ?`, statusPending, message, time.Now().Add(time.Duration(delaySeconds)*time.Second), id, d.workerID)
	return err
}

func (d *Dispatcher) retryCount(ctx context.Context, id int64) (int, error) {
	var retryCount int
	if err := d.conn.QueryRowCtx(ctx, &retryCount, `
SELECT retry_count
FROM interaction_outbox_events
WHERE id = ?`, id); err != nil {
		return 0, fmt.Errorf("query retry count: %w", err)
	}
	return retryCount, nil
}

func newWorkerID() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%s", hostname, uuid.NewString())
}
