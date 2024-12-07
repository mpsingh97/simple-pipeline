package tasks

import (
	"context"
	"fmt"
	"go-task/models"
	"go-task/postgres"
	"math/rand"
	"time"
)

type AssembleOrchestrator struct {
	models.DefaultTaskProcessor
}

func (i *AssembleOrchestrator) Process(ctx context.Context, m models.Milestone, pgClient *postgres.Client) error {
	if err := i.PreProcess(ctx, m, pgClient); err != nil {
		return err
	}

	time.Sleep(time.Duration(rand.Intn(10)) * time.Second) // business logic

	var pErr error
	if rand.Float64() < 0.2 {
		pErr = fmt.Errorf("oh no, a random error!")
	}

	if err := i.PostProcess(ctx, pErr, m, pgClient); err != nil {
		return err
	}
	return nil
}
