package models

import "github.com/google/uuid"

type RatingModel struct {
	ID       uuid.UUID `bson:"_id"`
	ReaderID uuid.UUID `bson:"reader_id"`
	BookID   uuid.UUID `bson:"book_id"`
	Review   string    `bson:"review"`
	Rating   int       `bson:"rating"`
}
