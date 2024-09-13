package models

import "github.com/google/uuid"

type BookModel struct {
	ID             uuid.UUID `bson:"_id"`
	Title          string    `bson:"title"`
	Author         string    `bson:"author"`
	Publisher      string    `bson:"publisher"`
	CopiesNumber   uint      `bson:"copies_number"`
	Rarity         string    `bson:"rarity"`
	Genre          string    `bson:"genre"`
	PublishingYear uint      `bson:"publishing_year"`
	Language       string    `bson:"language"`
	AgeLimit       uint      `bson:"age_limit"`
}
