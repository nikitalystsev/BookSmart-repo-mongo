package models

import "github.com/google/uuid"

type ReaderModel struct {
	ID          uuid.UUID `bson:"_id"`
	Fio         string    `bson:"fio"`
	PhoneNumber string    `bson:"phone_number"`
	Age         uint      `bson:"age"`
	Password    string    `bson:"password"`
	Role        string    `bson:"role"`
}
