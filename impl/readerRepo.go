package impl

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/nikitalystsev/BookSmart-services/core/models"
	"github.com/nikitalystsev/BookSmart-services/errs"
	"github.com/nikitalystsev/BookSmart-services/intfRepo"
	repomodels "github.com/nikitalystsev/Booksmart-repo-mongo/core/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type ReaderRepo struct {
	dbReader   *mongo.Collection
	dbFavorite *mongo.Collection
	client     *redis.Client
	logger     *logrus.Entry
}

func NewReaderRepo(db *mongo.Database, client *redis.Client, logger *logrus.Entry) intfRepo.IReaderRepo {
	return &ReaderRepo{
		dbReader:   db.Collection("reader"),
		dbFavorite: db.Collection("favorite_books"),
		client:     client,
		logger:     logger,
	}
}

func (rr *ReaderRepo) Create(ctx context.Context, reader *models.ReaderModel) error {
	rr.logger.Infof("inserting reader with ID: %s", reader.ID)

	_, err := rr.dbReader.InsertOne(ctx, rr.convertToRepoReaderModel(reader))
	if err != nil {
		rr.logger.Errorf("error inserting reader: %v", err)
		return err
	}

	rr.logger.Infof("inserted reader with ID: %s", reader.ID)

	return nil
}

func (rr *ReaderRepo) GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.ReaderModel, error) {
	rr.logger.Infof("find reader with phoneNumber: %s", phoneNumber)

	one := rr.dbReader.FindOne(ctx, bson.M{"phone_number": phoneNumber})

	if one.Err() != nil && !errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Errorf("error find reader by phoneNumber: %v", one.Err())
		return nil, one.Err()
	}
	if one.Err() != nil && errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Warnf("reader with this phoneNumber not found: %s", phoneNumber)
		return nil, errs.ErrReaderDoesNotExists
	}

	var reader repomodels.ReaderModel
	if err := one.Decode(&reader); err != nil {
		rr.logger.Errorf("error decoding reader: %v", err)
		return nil, err
	}

	rr.logger.Infof("found reader with phoneNumber: %s", phoneNumber)

	return rr.convertToReaderModel(&reader), nil
}

func (rr *ReaderRepo) GetByID(ctx context.Context, ID uuid.UUID) (*models.ReaderModel, error) {
	rr.logger.Infof("find reader with ID: %s", ID)

	one := rr.dbReader.FindOne(ctx, bson.M{"_id": ID})

	if one.Err() != nil && !errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Errorf("error find reader with ID: %v", one.Err())
		return nil, one.Err()
	}
	if one.Err() != nil && errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Warnf("reader with this ID not found: %v", ID)
		return nil, errs.ErrReaderDoesNotExists
	}

	var reader repomodels.ReaderModel
	if err := one.Decode(&reader); err != nil {
		rr.logger.Errorf("error decoding reader: %v", err)
		return nil, err
	}

	rr.logger.Infof("found reader with ID: %s", ID)

	return rr.convertToReaderModel(&reader), nil
}

func (rr *ReaderRepo) IsFavorite(ctx context.Context, readerID, bookID uuid.UUID) (bool, error) {
	rr.logger.Infof("book with ID = %s already is favorite?", bookID)

	count, err := rr.dbFavorite.CountDocuments(ctx, bson.M{"reader_id": readerID, "book_id": bookID})
	if err != nil {
		rr.logger.Errorf("error checking favorite book: %v", err)
		return false, err
	}

	rr.logger.Infof("checked favorite book")

	return count > 0, nil
}

func (rr *ReaderRepo) AddToFavorites(ctx context.Context, readerID, bookID uuid.UUID) error {
	rr.logger.Infof("reader (ID = %s) adding book (ID = %s) to favorites", readerID, bookID)

	_, err := rr.dbFavorite.InsertOne(ctx, bson.M{"reader_id": readerID, "book_id": bookID})
	if err != nil {
		rr.logger.Errorf("error adding book to favorites: %v", err)
		return err
	}

	rr.logger.Infof("reader (ID = %s) added book (ID = %s) to favorites", readerID, bookID)

	return nil
}

func (rr *ReaderRepo) SaveRefreshToken(ctx context.Context, id uuid.UUID, token string, ttl time.Duration) error {
	rr.logger.Infof("saving refresh token in redis")

	err := rr.client.Set(ctx, token, id.String(), ttl).Err()
	if err != nil {
		rr.logger.Errorf("error saving refresh token: %v", err)
		return err
	}

	rr.logger.Infof("refresh token saved in redis")

	return nil
}

func (rr *ReaderRepo) GetByRefreshToken(ctx context.Context, token string) (*models.ReaderModel, error) {
	rr.logger.Infof("getting reader by refresh token: %s", token)

	var readerID uuid.UUID

	readerIDStr, err := rr.client.Get(ctx, token).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		rr.logger.Errorf("error getting reader by refresh token: %v", err)
		return nil, err
	}
	if err != nil && errors.Is(err, redis.Nil) {
		rr.logger.Errorf("reader with this refresh token not found: %s", token)
		return nil, errs.ErrReaderDoesNotExists
	}

	readerID, err = uuid.Parse(readerIDStr)
	if err != nil {
		rr.logger.Errorf("error parsing readerID by refresh token: %v", err)
		return nil, err
	}

	one := rr.dbReader.FindOne(ctx, bson.M{"_id": readerID})

	if one.Err() != nil && !errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Errorf("error find reader with ID: %v", one.Err())
		return nil, one.Err()
	}
	if one.Err() != nil && errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Warnf("reader with this ID not found: %v", readerID)
		return nil, errs.ErrReaderDoesNotExists
	}

	var reader repomodels.ReaderModel
	if err = one.Decode(&reader); err != nil {
		rr.logger.Errorf("error decoding reader: %v", err)
		return nil, err
	}

	rr.logger.Infof("getting reader by refresh token: %v", token)

	return rr.convertToReaderModel(&reader), nil
}

func (rr *ReaderRepo) convertToReaderModel(reader *repomodels.ReaderModel) *models.ReaderModel {
	return &models.ReaderModel{
		ID:          reader.ID,
		Fio:         reader.Fio,
		PhoneNumber: reader.PhoneNumber,
		Age:         reader.Age,
		Password:    reader.Password,
		Role:        reader.Role,
	}
}

func (rr *ReaderRepo) convertToRepoReaderModel(reader *models.ReaderModel) *repomodels.ReaderModel {
	return &repomodels.ReaderModel{
		ID:          reader.ID,
		Fio:         reader.Fio,
		PhoneNumber: reader.PhoneNumber,
		Age:         reader.Age,
		Password:    reader.Password,
		Role:        reader.Role,
	}
}
