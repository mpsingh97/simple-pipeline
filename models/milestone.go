package models

import (
	"os"
	"time"

	uuid "github.com/satori/go.uuid"
)

const MAX_RETRIES_ALLOWED = 3

type Milestone struct {
	TaskID            string
	IngestStatus      TaskStatus
	TranscodeStatus   TaskStatus
	MetadataGenStatus TaskStatus
	AssembleStatus    TaskStatus
	PublishStatus     TaskStatus
	OverriddenBy      string
	HostMachine       string
	ProcessID         int
	Retries           int
	StartTime         time.Time
	EndTime           time.Time
	ErrMsg            string
}

func NewMilestone() Milestone {
	hostname, _ := os.Hostname()

	m := Milestone{
		TaskID:            uuid.NewV4().String(),
		IngestStatus:      PENDING,
		TranscodeStatus:   PENDING,
		MetadataGenStatus: PENDING,
		AssembleStatus:    PENDING,
		PublishStatus:     PENDING,
		OverriddenBy:      "",
		HostMachine:       hostname,
		ProcessID:         os.Getpid(),
		Retries:           0,
		StartTime:         time.Now().UTC(),
		EndTime:           time.Date(2099, time.Month(1), 1, 0, 0, 0, 0, time.UTC),
		ErrMsg:            "",
	}
	return m
}
