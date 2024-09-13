package models

import (
	"github.com/google/uuid"
	"time"
)

type LibCardModel struct {
	ID           uuid.UUID `bson:"_id"`
	ReaderID     uuid.UUID `bson:"reader_id"`
	LibCardNum   string    `bson:"lib_card_num"`
	Validity     int       `bson:"validity"`
	IssueDate    time.Time `bson:"issue_date"`
	ActionStatus bool      `bson:"action_status"`
}
