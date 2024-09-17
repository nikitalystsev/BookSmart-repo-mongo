package impl

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	repomodels "github.com/nikitalystsev/BookSmart-repo-mongo/core/models"
	"github.com/nikitalystsev/BookSmart-services/core/models"
	"github.com/nikitalystsev/BookSmart-services/errs"
	"github.com/nikitalystsev/BookSmart-services/intfRepo"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type RatingRepo struct {
	db     *mongo.Collection
	logger *logrus.Entry
}

func NewRatingRepo(db *mongo.Database, logger *logrus.Entry) intfRepo.IRatingRepo {
	return &RatingRepo{db: db.Collection("rating"), logger: logger}
}

func (rr *RatingRepo) Create(ctx context.Context, rating *models.RatingModel) error {
	rr.logger.Infof("inserting rating with ID: %s", rating.ID)

	_, err := rr.db.InsertOne(ctx, rr.convertToRepoRatingModel(rating))
	if err != nil {
		rr.logger.Errorf("error inserting rating: %v", err)
		return err
	}

	rr.logger.Infof("inserted book with ID: %s", rating.ID)

	return nil
}

func (rr *RatingRepo) GetByReaderAndBook(ctx context.Context, readerID, bookID uuid.UUID) (*models.RatingModel, error) {
	rr.logger.Infof("find rating with readerID и bookID: %s и %s", readerID, bookID)

	one := rr.db.FindOne(ctx, bson.M{"reader_id": readerID, "book_id": bookID})

	if one.Err() != nil && !errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Errorf("error selecting rating: %v", one.Err())
		return nil, one.Err()
	}
	if one.Err() != nil && errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Warnf("rating with this readerID и bookID not found: %s и %s", readerID, bookID)
		return nil, errs.ErrRatingDoesNotExists
	}

	var rating repomodels.RatingModel
	if err := one.Decode(&rating); err != nil {
		rr.logger.Errorf("error decoding rating: %v", err)
		return nil, err
	}

	rr.logger.Infof("found rating with readerID и bookID: %s и %s", readerID, bookID)

	return rr.convertToRatingModel(&rating), nil
}

func (rr *RatingRepo) GetByBookID(ctx context.Context, bookID uuid.UUID) ([]*models.RatingModel, error) {
	rr.logger.Infof("find ratings with bookID: %s", bookID)

	filter := bson.M{
		"book_id": bookID,
	}

	cursor, err := rr.db.Find(ctx, filter)
	if err != nil {
		rr.logger.Errorf("error find ratings: %v", err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err = cursor.Close(ctx)
		if err != nil {
			fmt.Println("error close cursor")
		}
	}(cursor, ctx)

	var coreRatings []*repomodels.RatingModel
	if err = cursor.All(ctx, &coreRatings); err != nil {
		rr.logger.Printf("error decoding ratings: %v", err)
		return nil, err
	}

	if len(coreRatings) == 0 {
		rr.logger.Warnf("ratings with this bookID not found: %s", bookID)
		return nil, errs.ErrRatingDoesNotExists
	}

	rr.logger.Infof("found ratings with bookID: %s", bookID)

	ratings := make([]*models.RatingModel, len(coreRatings))
	for i, coreReservation := range coreRatings {
		ratings[i] = rr.convertToRatingModel(coreReservation)
	}

	return ratings, nil
}

func (rr *RatingRepo) convertToRatingModel(rating *repomodels.RatingModel) *models.RatingModel {
	return &models.RatingModel{
		ID:       rating.ID,
		ReaderID: rating.ReaderID,
		BookID:   rating.BookID,
		Review:   rating.Review,
		Rating:   rating.Rating,
	}
}

func (rr *RatingRepo) convertToRepoRatingModel(rating *models.RatingModel) *repomodels.RatingModel {
	return &repomodels.RatingModel{
		ID:       rating.ID,
		ReaderID: rating.ReaderID,
		BookID:   rating.BookID,
		Review:   rating.Review,
		Rating:   rating.Rating,
	}
}
