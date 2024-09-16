package models

import "github.com/google/uuid"

type RatingModel struct {
	ID       uuid.UUID `db:"id"`
	ReaderID uuid.UUID `db:"reader_id"`
	BookID   uuid.UUID `db:"book_id"`
	Review   string    `db:"review"`
	Rating   int       `db:"rating"`
}
