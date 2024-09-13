package impl

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	repomodels "github.com/nikitalystsev/BookSmart-repo-mongo/core/models"
	"github.com/nikitalystsev/BookSmart-services/core/models"
	"github.com/nikitalystsev/BookSmart-services/errs"
	"github.com/nikitalystsev/BookSmart-services/impl"
	"github.com/nikitalystsev/BookSmart-services/intfRepo"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type ReservationRepo struct {
	db     *mongo.Collection
	logger *logrus.Entry
}

func NewReservationRepo(db *mongo.Database, logger *logrus.Entry) intfRepo.IReservationRepo {
	return &ReservationRepo{db: db.Collection("reservation"), logger: logger}
}

func (rr *ReservationRepo) Create(ctx context.Context, reservation *models.ReservationModel) error {
	rr.logger.Infof("inserting reservation with ID: %s", reservation.ID)

	_, err := rr.db.InsertOne(ctx, rr.convertToRepoReservationModel(reservation))
	if err != nil {
		rr.logger.Errorf("error inserting reservation: %v", err)
		return err
	}

	rr.logger.Infof("inserted reservation with ID: %s", reservation.ID)

	return nil
}

func (rr *ReservationRepo) GetByReaderAndBook(ctx context.Context, readerID, bookID uuid.UUID) (*models.ReservationModel, error) {
	rr.logger.Infof("find reservation with readerID и bookID: %s и %s", readerID, bookID)

	if err := rr.updateReservationStates(ctx); err != nil {
		rr.logger.Errorf("error updating reservations status: %v", err)
		return nil, err
	}

	one := rr.db.FindOne(ctx, bson.M{"reader_id": readerID, "book_id": bookID})

	if one.Err() != nil && !errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Errorf("error selecting reservation: %v", one.Err())
		return nil, one.Err()
	}
	if one.Err() != nil && errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Warnf("reservation with this readerID и bookID not found: %s и %s", readerID, bookID)
		return nil, errs.ErrReservationDoesNotExists
	}

	var reservation repomodels.ReservationModel
	if err := one.Decode(&reservation); err != nil {
		rr.logger.Errorf("error decoding reservation: %v", err)
		return nil, err
	}

	rr.logger.Infof("found reservation with readerID и bookID: %s и %s", readerID, bookID)

	return rr.convertToReservationModel(&reservation), nil
}

func (rr *ReservationRepo) GetByID(ctx context.Context, ID uuid.UUID) (*models.ReservationModel, error) {
	rr.logger.Infof("find reservation with ID: %s", ID)

	if err := rr.updateReservationStates(ctx); err != nil {
		rr.logger.Errorf("error updating reservations status: %v", err)
		return nil, err
	}

	one := rr.db.FindOne(ctx, bson.M{"_id": ID})

	if one.Err() != nil && !errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Errorf("error find reservation: %v", one.Err())
		return nil, one.Err()
	}
	if one.Err() != nil && errors.Is(one.Err(), mongo.ErrNoDocuments) {
		rr.logger.Warnf("reservation with this ID not found: %s", ID)
		return nil, errs.ErrReservationDoesNotExists
	}

	var reservation repomodels.ReservationModel
	if err := one.Decode(&reservation); err != nil {
		rr.logger.Errorf("error decoding reservation: %v", err)
		return nil, err
	}

	rr.logger.Infof("found reservation with ID: %s", ID)

	return rr.convertToReservationModel(&reservation), nil
}

// GetByBookID TODO добавить в схемы
func (rr *ReservationRepo) GetByBookID(ctx context.Context, bookID uuid.UUID) ([]*models.ReservationModel, error) {
	rr.logger.Infof("find reservation with bookID: %s", bookID)

	if err := rr.updateReservationStates(ctx); err != nil {
		rr.logger.Errorf("error updating reservations status: %v", err)
		return nil, err
	}

	filter := bson.M{
		"book_id": bookID,
	}

	cursor, err := rr.db.Find(ctx, filter)
	if err != nil {
		rr.logger.Errorf("error find expired reservations: %v", err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err = cursor.Close(ctx)
		if err != nil {
			fmt.Println("error close cursor")
		}
	}(cursor, ctx)

	var coreReservations []*repomodels.ReservationModel
	if err = cursor.All(ctx, &coreReservations); err != nil {
		rr.logger.Printf("error decoding reservations: %v", err)
		return nil, err
	}

	if len(coreReservations) == 0 {
		rr.logger.Warnf("reservations with this bookID not found: %s", bookID)
		return nil, errs.ErrReservationDoesNotExists
	}

	rr.logger.Infof("found reservation with bookID: %s", bookID)

	reservations := make([]*models.ReservationModel, len(coreReservations))
	for i, coreReservation := range coreReservations {
		reservations[i] = rr.convertToReservationModel(coreReservation)
	}

	return reservations, nil
}

func (rr *ReservationRepo) Update(ctx context.Context, reservation *models.ReservationModel) error {
	rr.logger.Infof("updating reservation with ID: %s", reservation.ID)

	updateData := bson.M{
		"$set": bson.M{
			"reader_id":   reservation.ReaderID,
			"book_id":     reservation.BookID,
			"issue_date":  reservation.IssueDate,
			"return_date": reservation.ReturnDate,
			"state":       reservation.State,
		},
	}

	one, err := rr.db.UpdateOne(ctx, bson.M{"_id": reservation.ID}, updateData)
	if err != nil {
		rr.logger.Errorf("error updating reservation with ID: %v", err)
		return err
	}

	if one.MatchedCount == 0 {
		rr.logger.Warnf("reservation with this ID not found: %v", reservation.ID)
		return errs.ErrReservationDoesNotExists
	}

	rr.logger.Infof("updated reservation with ID: %s", reservation.ID)

	return nil
}

func (rr *ReservationRepo) GetExpiredByReaderID(ctx context.Context, readerID uuid.UUID) ([]*models.ReservationModel, error) {
	rr.logger.Infof("find expired reservations with readerID: %s", readerID)

	if err := rr.updateReservationStates(ctx); err != nil {
		rr.logger.Errorf("error updating reservations status: %v", err)
		return nil, err
	}

	filter := bson.M{
		"$or": []bson.M{
			{"return_date": bson.M{"$lte": time.Now()}},
			{"state": "Expired"},
		},
		"reader_id": readerID,
	}

	cursor, err := rr.db.Find(ctx, filter)
	if err != nil {
		rr.logger.Errorf("error find expired reservations: %v", err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err = cursor.Close(ctx)
		if err != nil {
			fmt.Println("error close cursor")
		}
	}(cursor, ctx)

	var coreReservations []*repomodels.ReservationModel
	if err = cursor.All(ctx, &coreReservations); err != nil {
		rr.logger.Printf("error decoding reservations: %v", err)
		return nil, err
	}

	if len(coreReservations) == 0 {
		rr.logger.Warnf("expired reservations with this readerID not found: %s", readerID)
		return nil, errs.ErrReservationDoesNotExists
	}

	rr.logger.Infof("found %d expired reservations with readerID %s", len(coreReservations), readerID)

	reservations := make([]*models.ReservationModel, len(coreReservations))
	for i, coreReservation := range coreReservations {
		reservations[i] = rr.convertToReservationModel(coreReservation)
	}

	return reservations, nil
}

func (rr *ReservationRepo) GetActiveByReaderID(ctx context.Context, readerID uuid.UUID) ([]*models.ReservationModel, error) {
	rr.logger.Infof("find active reservations with readerID: %s", readerID)

	if err := rr.updateReservationStates(ctx); err != nil {
		rr.logger.Errorf("error updating reservations status: %v", err)
		return nil, err
	}
	filter := bson.M{
		"reader_id": readerID,
		"state": bson.M{
			"$nin": []string{impl.ReservationExpired, impl.ReservationClosed},
		},
	}

	cursor, err := rr.db.Find(ctx, filter)
	if err != nil {
		rr.logger.Errorf("error find active reservations: %v", err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err = cursor.Close(ctx)
		if err != nil {
			fmt.Println("error close cursor")
		}
	}(cursor, ctx)

	var coreReservations []*repomodels.ReservationModel
	if err = cursor.All(ctx, &coreReservations); err != nil {
		rr.logger.Printf("error decoding reservations: %v", err)
		return nil, err
	}

	if len(coreReservations) == 0 {
		rr.logger.Warnf("active reservations with this readerID not found: %s", readerID)
		return nil, errs.ErrReservationDoesNotExists
	}

	rr.logger.Infof("found %d active reservations with readerID %s", len(coreReservations), readerID)

	reservations := make([]*models.ReservationModel, len(coreReservations))
	for i, coreReservation := range coreReservations {
		reservations[i] = rr.convertToReservationModel(coreReservation)
	}

	return reservations, nil
}

func (rr *ReservationRepo) updateReservationStates(ctx context.Context) error {
	filterExpired := bson.M{
		"state":      bson.M{"$in": []string{impl.ReservationIssued, impl.ReservationExtended}},
		"returnDate": bson.M{"$lt": time.Now()},
	}
	updateExpired := bson.M{
		"$set": bson.M{"state": impl.ReservationExpired},
	}

	_, err := rr.db.UpdateMany(ctx, filterExpired, updateExpired)
	if err != nil {
		return err
	}

	return nil
}

func (rr *ReservationRepo) convertToRepoReservationModel(reservation *models.ReservationModel) *repomodels.ReservationModel {
	return &repomodels.ReservationModel{
		ID:         reservation.ID,
		ReaderID:   reservation.ReaderID,
		BookID:     reservation.BookID,
		IssueDate:  reservation.IssueDate,
		ReturnDate: reservation.ReturnDate,
		State:      reservation.State,
	}
}

func (rr *ReservationRepo) convertToReservationModel(reservation *repomodels.ReservationModel) *models.ReservationModel {
	return &models.ReservationModel{
		ID:         reservation.ID,
		ReaderID:   reservation.ReaderID,
		BookID:     reservation.BookID,
		IssueDate:  reservation.IssueDate,
		ReturnDate: reservation.ReturnDate,
		State:      reservation.State,
	}
}
