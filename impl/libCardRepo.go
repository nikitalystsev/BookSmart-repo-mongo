package impl

import (
	"context"
	"errors"
	"github.com/google/uuid"
	repomodels "github.com/nikitalystsev/BookSmart-repo-mongo/core/models"
	"github.com/nikitalystsev/BookSmart-services/core/models"
	"github.com/nikitalystsev/BookSmart-services/errs"
	"github.com/nikitalystsev/BookSmart-services/intfRepo"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type LibCardRepo struct {
	db     *mongo.Collection
	logger *logrus.Entry
}

func NewLibCardRepo(db *mongo.Database, logger *logrus.Entry) intfRepo.ILibCardRepo {
	return &LibCardRepo{db: db.Collection("lib_card"), logger: logger}
}

func (lcr *LibCardRepo) Create(ctx context.Context, libCard *models.LibCardModel) error {
	lcr.logger.Infof("inserting libCard with ID: %s", libCard.ID)

	_, err := lcr.db.InsertOne(ctx, lcr.convertToRepoLibCardModel(libCard))
	if err != nil {
		lcr.logger.Errorf("error inserting libCard: %v", err)
		return err
	}

	lcr.logger.Infof("inserted libCard with ID: %s", libCard.ID)

	return nil
}

func (lcr *LibCardRepo) GetByReaderID(ctx context.Context, readerID uuid.UUID) (*models.LibCardModel, error) {
	lcr.logger.Infof("find libCard with readerID: %s", readerID)

	if err := lcr.updateActionStatus(ctx); err != nil {
		lcr.logger.Errorf("error updating libCard status: %v", err)
		return nil, err
	}

	one := lcr.db.FindOne(ctx, bson.M{"reader_id": readerID})

	if one.Err() != nil && !errors.Is(one.Err(), mongo.ErrNoDocuments) {
		lcr.logger.Errorf("error find libCard: %v", one.Err())
		return nil, one.Err()
	}
	if one.Err() != nil && errors.Is(one.Err(), mongo.ErrNoDocuments) {
		lcr.logger.Warnf("libCard with this readerID not found: %v", readerID)
		return nil, errs.ErrLibCardDoesNotExists
	}

	var libCard repomodels.LibCardModel
	if err := one.Decode(&libCard); err != nil {
		lcr.logger.Errorf("error decoding libCard: %v", err)
		return nil, err
	}

	lcr.logger.Infof("found libCard with readerID: %s", readerID)

	return lcr.convertToLibCardModel(&libCard), nil
}

func (lcr *LibCardRepo) GetByNum(ctx context.Context, libCardNum string) (*models.LibCardModel, error) {
	lcr.logger.Infof("find libCard with num: %s", libCardNum)

	if err := lcr.updateActionStatus(ctx); err != nil {
		lcr.logger.Errorf("error updating libCard status: %v", err)
		return nil, err
	}

	one := lcr.db.FindOne(ctx, bson.M{"lib_card_num": libCardNum})

	if one.Err() != nil && !errors.Is(one.Err(), mongo.ErrNoDocuments) {
		lcr.logger.Errorf("error find libCard: %v", one.Err())
		return nil, one.Err()
	}
	if one.Err() != nil && errors.Is(one.Err(), mongo.ErrNoDocuments) {
		lcr.logger.Warnf("libCard with this num not found: %v", libCardNum)
		return nil, errs.ErrLibCardDoesNotExists
	}

	var libCard repomodels.LibCardModel
	if err := one.Decode(&libCard); err != nil {
		lcr.logger.Errorf("error decoding libCard: %v", err)
		return nil, err
	}

	lcr.logger.Infof("found libCard with num: %s", libCardNum)

	return lcr.convertToLibCardModel(&libCard), nil
}

func (lcr *LibCardRepo) Update(ctx context.Context, libCard *models.LibCardModel) error {
	lcr.logger.Infof("updating libCard with ID: %s", libCard.ID)

	updateData := bson.M{
		"$set": bson.M{
			"reader_id":     libCard.ReaderID,
			"lib_card_num":  libCard.LibCardNum,
			"validity":      libCard.Validity,
			"issue_date":    libCard.IssueDate,
			"action_status": libCard.ActionStatus,
		},
	}

	one, err := lcr.db.UpdateOne(ctx, bson.M{"_id": libCard.ID}, updateData)
	if err != nil {
		lcr.logger.Errorf("error updating libCard: %v", err)
		return err
	}

	if one.MatchedCount == 0 {
		lcr.logger.Warnf("libCard with this ID not found: %v", libCard.ID)
		return errs.ErrLibCardDoesNotExists
	}

	lcr.logger.Infof("updated libCard with ID: %s", libCard.ID)

	return nil
}

func (lcr *LibCardRepo) updateActionStatus(ctx context.Context) error {
	filter := bson.M{
		"action_status": true,
		"issue_date":    bson.M{"$lt": time.Now().AddDate(0, 0, -365)},
	}
	update := bson.M{
		"$set": bson.M{"action_status": false},
	}

	_, err := lcr.db.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (lcr *LibCardRepo) convertToLibCardModel(libCard *repomodels.LibCardModel) *models.LibCardModel {
	return &models.LibCardModel{
		ID:           libCard.ID,
		ReaderID:     libCard.ReaderID,
		LibCardNum:   libCard.LibCardNum,
		Validity:     libCard.Validity,
		IssueDate:    libCard.IssueDate,
		ActionStatus: libCard.ActionStatus,
	}
}

func (lcr *LibCardRepo) convertToRepoLibCardModel(libCard *models.LibCardModel) *repomodels.LibCardModel {
	return &repomodels.LibCardModel{
		ID:           libCard.ID,
		ReaderID:     libCard.ReaderID,
		LibCardNum:   libCard.LibCardNum,
		Validity:     libCard.Validity,
		IssueDate:    libCard.IssueDate,
		ActionStatus: libCard.ActionStatus,
	}
}
