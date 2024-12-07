package main

import (
	"context"
	"fmt"
	"go-task/models"
	"go-task/postgres"
	"go-task/tasks"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Netflix/go-env"
)

type Environment struct {
	PRIMARY_DB_URL  string `env:"PRIMARY_DB_URL,required=true"`
	REPLICA_DB_URLS string `env:"REPLICA_DB_URLS,required=true"`
}

func main() {
	var environment Environment
	_, err := env.UnmarshalFromEnviron(&environment)
	if err != nil {
		log.Fatal(err)
	}

	pgClient, err := postgres.New(environment.PRIMARY_DB_URL, strings.Split(environment.REPLICA_DB_URLS, ","))
	if err != nil {
		log.Fatal(err)
	}
	defer pgClient.Close()

	orchestrators := []models.TaskScheduler{
		{
			TaskProcessor: &tasks.IngestOrchestrator{
				DefaultTaskProcessor: models.DefaultTaskProcessor{
					Task: models.Task{
						For: models.INGEST,
					},
				},
			},
			Interval: time.Second * 5,
			PgClient: pgClient,
		},
		{
			TaskProcessor: &tasks.TranscodeOrchestrator{
				DefaultTaskProcessor: models.DefaultTaskProcessor{
					Task: models.Task{
						For:          models.TRANSCODE,
						Dependencies: []models.TaskType{models.INGEST},
					},
				},
			},
			Interval: time.Second * 5,
			PgClient: pgClient,
		},
		{
			TaskProcessor: &tasks.MetadataGenOrchestrator{
				DefaultTaskProcessor: models.DefaultTaskProcessor{
					Task: models.Task{
						For:          models.METADATA_GEN,
						Dependencies: []models.TaskType{models.INGEST},
					},
				},
			},
			Interval: time.Second * 5,
			PgClient: pgClient,
		},
		{
			TaskProcessor: &tasks.AssembleOrchestrator{
				DefaultTaskProcessor: models.DefaultTaskProcessor{
					Task: models.Task{
						For:          models.ASSEMBLE,
						Dependencies: []models.TaskType{models.TRANSCODE, models.METADATA_GEN},
					},
				},
			},
			Interval: time.Second * 5,
			PgClient: pgClient,
		},
		{
			TaskProcessor: &tasks.PublishOrchestrator{
				DefaultTaskProcessor: models.DefaultTaskProcessor{
					Task: models.Task{
						For:          models.PUBLISH,
						Dependencies: []models.TaskType{models.ASSEMBLE},
					},
				},
			},
			Interval: time.Second * 5,
			PgClient: pgClient,
		},
	}

	for _, orch := range orchestrators {
		go orch.Run(context.Background())
	}

	const insertQuery = "INSERT INTO tasks VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', %d, %d, '%s', '%s', '%s');"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		m := models.NewMilestone()
		query := fmt.Sprintf(
			insertQuery,
			m.TaskID, m.IngestStatus, m.TranscodeStatus, m.MetadataGenStatus, m.AssembleStatus, m.PublishStatus, m.OverriddenBy,
			m.HostMachine, m.ProcessID, m.Retries, m.StartTime.Format("2006-01-02 15:04:05"), m.EndTime.Format("2006-01-02 15:04:05"), m.ErrMsg,
		)
		conn, err := pgClient.PrimaryPool.Acquire(context.Background())
		if err != nil {
			log.Printf("error acquiring connection: %v", err)
			return
		}
		defer conn.Release()
		if _, err := conn.Query(context.Background(), query); err != nil {
			log.Printf("error creating milestone in database: %v", err)
			return
		}
	})

	log.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Println("Error starting server:", err)
	}
}
