package models

import (
	"context"
	"errors"
	"fmt"
	"go-task/postgres"
	"log"
	"os"
	"strings"
	"time"
)

const MarkSuccessQuery = "UPDATE tasks SET %s = '%s', error_message = '', end_time = '%s', retries = 0 WHERE task_id = '%s';"
const MarkFailQuery = "UPDATE tasks SET %s = '%s', error_message = '%s', end_time = '%s', retries = retries + 1 WHERE task_id = '%s';"

type TaskType string

const (
	INGEST       TaskType = "ingest_status"
	TRANSCODE    TaskType = "transcode_status"
	METADATA_GEN TaskType = "metadata_gen_status"
	ASSEMBLE     TaskType = "assemble_status"
	PUBLISH      TaskType = "publish_status"
)

type TaskStatus string

const (
	PENDING    TaskStatus = "pending"
	FAILED     TaskStatus = "failed"
	COMPLETED  TaskStatus = "completed"
	OVERRIDDEN TaskStatus = "overridden"
	INPROGRESS TaskStatus = "inprogress"
)

type Task struct {
	For          TaskType
	Dependencies []TaskType
}

type TaskProcessor interface {
	PreProcess(context.Context, Milestone, *postgres.Client) error
	Process(context.Context, Milestone, *postgres.Client) error
	PostProcess(context.Context, error, Milestone, *postgres.Client) error
	Poll(context.Context, time.Duration, *postgres.Client) <-chan Milestone
	GetFor() TaskType
}

type TaskScheduler struct {
	TaskProcessor TaskProcessor
	Interval      time.Duration
	PgClient      *postgres.Client
}

func (t *TaskScheduler) Run(ctx context.Context) {
	milestones := t.TaskProcessor.Poll(ctx, t.Interval, t.PgClient)
	for {
		select {
		case m, ok := <-milestones:
			if !ok {
				log.Printf("[%v] polling stopped", t.TaskProcessor.GetFor())
				return
			}
			go func(m Milestone) {
				if err := t.TaskProcessor.Process(ctx, m, t.PgClient); err != nil {
					log.Printf("[%v] error processing milestone %s: %v", t.TaskProcessor.GetFor(), m.TaskID, err)
				} else {
					log.Printf("[%v] successfully processed milestone %s", t.TaskProcessor.GetFor(), m.TaskID)
				}
			}(m)
		}
	}
}

type DefaultTaskProcessor struct {
	Task
}

func (d DefaultTaskProcessor) PreProcess(ctx context.Context, m Milestone, pgClient *postgres.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	hostname, _ := os.Hostname()
	query := fmt.Sprintf(
		"UPDATE tasks SET %s = '%s', host_machine = '%s', process_id = %d, start_time = '%s' WHERE task_id = (SELECT task_id FROM tasks WHERE task_id = '%s' AND (%s = '%s' OR (%s = '%s' AND retries < %d)) FOR UPDATE SKIP LOCKED) RETURNING *;",
		d.For, INPROGRESS, hostname, os.Getpid(), time.Now().UTC().Format("2006-01-02 15:04:05"),
		m.TaskID, d.For, PENDING, d.For, FAILED, MAX_RETRIES_ALLOWED,
	)
	conn, err := pgClient.PrimaryPool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("picked up by another goroutine")
	}
	return nil
}

func (d DefaultTaskProcessor) PostProcess(ctx context.Context, pErr error, m Milestone, pgClient *postgres.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := pgClient.PrimaryPool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if pErr != nil {
		if _, err := conn.Query(ctx, fmt.Sprintf(MarkFailQuery, d.For, FAILED, pErr, time.Now().UTC().Format("2006-01-02 15:04:05"), m.TaskID)); err != nil {
			return err
		}
	} else {
		if _, err := conn.Query(ctx, fmt.Sprintf(MarkSuccessQuery, d.For, COMPLETED, time.Now().UTC().Format("2006-01-02 15:04:05"), m.TaskID)); err != nil {
			return err
		}
	}
	return nil
}

func (d DefaultTaskProcessor) Poll(ctx context.Context, interval time.Duration, pgClient *postgres.Client) <-chan Milestone {
	ch := make(chan Milestone, 100)

	conditions := fmt.Sprintf("(%s = '%s' AND retries < %d) OR ", d.For, FAILED, MAX_RETRIES_ALLOWED)
	dependencyConditions := []string{}
	for _, d := range d.Dependencies {
		dependencyConditions = append(dependencyConditions, fmt.Sprintf("%s = '%s'", d, COMPLETED))
	}
	if len(dependencyConditions) > 0 {
		conditions += fmt.Sprintf("(%s = '%s' AND (%s))", d.For, PENDING, strings.Join(dependencyConditions, " AND "))
	} else {
		conditions += fmt.Sprintf("%s = '%s'", d.For, PENDING)
	}

	query := fmt.Sprintf("SELECT * FROM tasks WHERE %s;", conditions)

	go func() {
		defer close(ch)
		for {
			time.Sleep(interval)

			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			conn, err := pgClient.GetReplicaPool().Acquire(ctx)
			if err != nil && errors.Is(err, context.DeadlineExceeded) {
				continue
			} else if err != nil {
				log.Printf("[%v] error in Poll: %v", d.For, err)
				continue
			}
			defer conn.Release()
			rows, err := conn.Query(ctx, query)
			if err != nil && errors.Is(err, context.DeadlineExceeded) {
				log.Printf("[%v] no data available for processing in Poll, going into deep sleep", d.For)
				time.Sleep(interval)
				continue
			} else if err != nil {
				log.Printf("[%v] error in Poll: %v", d.For, err)
				continue
			}

			numColumns := len(rows.FieldDescriptions())
			for rows.Next() {
				rowValues := make([]interface{}, numColumns)
				for i := range rowValues {
					rowValues[i] = new(interface{})
				}
				if err := rows.Scan(rowValues...); err != nil {
					log.Printf("[%v] error for query '%s': %v", d.For, query, err)
					break
				}
				var m Milestone
				err := Decode(rowValues, &m)
				if err != nil {
					log.Printf("error decoding into milestone for %v\n", rowValues...)
					break
				}
				ch <- m
			}

			rows.Close()
		}
	}()

	return ch
}

func (d DefaultTaskProcessor) GetFor() TaskType {
	return d.For
}
