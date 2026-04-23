package outbox

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	BatchSize           int `json:",default=50"`
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
		batchSize = 50
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
	logx.Info("article outbox dispatcher started")
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
		logx.Errorf("load article outbox events failed: %v", err)
		return
	}

	for _, event := range pending {
		if ctx.Err() != nil {
			return
		}

		held, err := d.extendLease(ctx, event.ID)
		if err != nil {
			logx.Errorf("extend article outbox event id=%d lease failed: %v", event.ID, err)
			continue
		}
		if !held {
			logx.Errorf("skip article outbox event id=%d because lease was lost", event.ID)
			continue
		}

		stopLeaseHeartbeat := d.startLeaseHeartbeat(ctx, event.ID)
		if err := d.publish(ctx, event); err != nil {
			stopLeaseHeartbeat()
			logx.Errorf("publish article outbox event id=%d topic=%s failed: %v", event.ID, event.Topic, err)
			if markErr := d.markFailed(ctx, event.ID, err); markErr != nil {
				logx.Errorf("mark article outbox event id=%d failed: %v", event.ID, markErr)
			}
			continue
		}
		stopLeaseHeartbeat()

		if err := d.markSent(ctx, event.ID); err != nil {
			logx.Errorf("mark article outbox event id=%d sent failed: %v", event.ID, err)
		}
	}
}

func (d *Dispatcher) claimPending(ctx context.Context) ([]Event, error) {
	var pending []Event
	err := d.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		txConn := sqlx.NewSqlConnFromSession(session)
		if err := txConn.QueryRowsCtx(ctx, &pending, `
SELECT id, topic, event_key, CAST(payload AS CHAR) AS payload
FROM article_outbox_events
WHERE
  (status = ? AND next_retry_at <= NOW())
  OR (status = ? AND locked_until < NOW())
ORDER BY id
LIMIT ?
FOR UPDATE SKIP LOCKED`, statusPending, statusProcessing, d.batchSize); err != nil {
			return err
		}
		if len(pending) == 0 {
			return nil
		}

		query, args := d.buildClaimQuery("article_outbox_events", pending, time.Now().Add(d.claimTTL))
		_, err := session.ExecCtx(ctx, query, args...)
		return err
	})
	return pending, err
}

func (d *Dispatcher) buildClaimQuery(table string, events []Event, lockedUntil time.Time) (string, []any) {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(events)), ",")
	args := make([]any, 0, len(events)+3)
	args = append(args, statusProcessing, d.workerID, lockedUntil)
	for _, event := range events {
		args = append(args, event.ID)
	}

	return fmt.Sprintf(`
UPDATE %s
SET status = ?, locked_by = ?, locked_until = ?, update_time = NOW()
WHERE id IN (%s)`, table, placeholders), args
}

func (d *Dispatcher) publish(ctx context.Context, event Event) error {
	producer, ok := d.producers[event.Topic]
	if !ok {
		producer = eventstream.NewProducer(eventstream.KafkaConf{Brokers: d.brokers}, event.Topic)
		d.producers[event.Topic] = producer
	}
	return producer.Publish(ctx, event.EventKey, []byte(event.Payload))
}

func (d *Dispatcher) extendLease(ctx context.Context, id int64) (bool, error) {
	res, err := d.conn.ExecCtx(ctx, `
UPDATE article_outbox_events
SET locked_until = ?, update_time = NOW()
WHERE id = ? AND status = ? AND locked_by = ?`, time.Now().Add(d.claimTTL), id, statusProcessing, d.workerID)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (d *Dispatcher) startLeaseHeartbeat(ctx context.Context, id int64) func() {
	interval := d.claimTTL / 3
	if interval < time.Second {
		interval = time.Second
	}

	heartbeatCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				held, err := d.extendLease(heartbeatCtx, id)
				if err != nil {
					logx.Errorf("extend article outbox event id=%d heartbeat failed: %v", id, err)
					continue
				}
				if !held {
					logx.Errorf("article outbox event id=%d lease lost during publish", id)
					return
				}
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}

func (d *Dispatcher) markSent(ctx context.Context, id int64) error {
	res, err := d.conn.ExecCtx(ctx, `
UPDATE article_outbox_events
SET status = ?, locked_by = '', locked_until = NULL, last_error = '', update_time = NOW()
WHERE id = ? AND status = ? AND locked_by = ?`, statusSent, id, statusProcessing, d.workerID)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("article outbox event %d lease lost before mark sent", id)
	}
	return nil
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

	res, err := d.conn.ExecCtx(ctx, `
UPDATE article_outbox_events
SET status = ?, retry_count = retry_count + 1, last_error = ?, next_retry_at = ?, locked_by = '', locked_until = NULL, update_time = NOW()
WHERE id = ? AND status = ? AND locked_by = ?`, statusPending, message, time.Now().Add(time.Duration(delaySeconds)*time.Second), id, statusProcessing, d.workerID)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("article outbox event %d lease lost before mark failed", id)
	}
	return nil
}

func (d *Dispatcher) retryCount(ctx context.Context, id int64) (int, error) {
	var retryCount int
	if err := d.conn.QueryRowCtx(ctx, &retryCount, `
SELECT retry_count
FROM article_outbox_events
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
