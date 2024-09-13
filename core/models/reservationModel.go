package models

import (
	"github.com/google/uuid"
	"time"
)

type ReservationModel struct {
	ID         uuid.UUID `bson:"_id"`
	ReaderID   uuid.UUID `bson:"reader_id"`
	BookID     uuid.UUID `bson:"book_id"`
	IssueDate  time.Time `bson:"issue_date"`
	ReturnDate time.Time `bson:"return_date"`
	State      string    `bson:"state"`
}
