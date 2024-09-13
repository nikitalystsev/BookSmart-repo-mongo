package impl

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/nikitalystsev/BookSmart-services/core/dto"
	"github.com/nikitalystsev/BookSmart-services/core/models"
	"github.com/nikitalystsev/BookSmart-services/errs"
	"github.com/nikitalystsev/BookSmart-services/intfRepo"
	repomodels "github.com/nikitalystsev/Booksmart-repo-mongo/core/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BookRepo struct {
	db     *mongo.Collection
	logger *logrus.Entry
}

func NewBookRepo(db *mongo.Database, logger *logrus.Entry) intfRepo.IBookRepo {
	return &BookRepo{db: db.Collection("book"), logger: logger}
}

func (br *BookRepo) Create(ctx context.Context, book *models.BookModel) error {
	br.logger.Infof("inserting book with ID: %s", book.ID)

	_, err := br.db.InsertOne(ctx, br.convertToRepoBookModel(book))
	if err != nil {
		br.logger.Errorf("error inserting book: %v", err)
		return err
	}

	br.logger.Infof("inserted book with ID: %s", book.ID)

	return nil
}

func (br *BookRepo) GetByID(ctx context.Context, ID uuid.UUID) (*models.BookModel, error) {
	br.logger.Infof("find book with ID: %s", ID)

	one := br.db.FindOne(ctx, bson.M{"_id": ID})
	if one.Err() != nil && !errors.Is(one.Err(), mongo.ErrNoDocuments) {
		br.logger.Errorf("error find book with ID: %v", one.Err())
		return nil, one.Err()
	}
	if one.Err() != nil && errors.Is(one.Err(), mongo.ErrNoDocuments) {
		br.logger.Warnf("book with this ID not found %s", ID)
		return nil, errs.ErrBookDoesNotExists
	}

	var book repomodels.BookModel
	if err := one.Decode(&book); err != nil {
		br.logger.Errorf("error decoding book: %v", err)
		return nil, err
	}

	br.logger.Infof("found book with ID: %s", ID)

	return br.convertToBookModel(&book), nil
}

func (br *BookRepo) GetByTitle(ctx context.Context, title string) (*models.BookModel, error) {
	br.logger.Infof("find book by title: %s", title)

	one := br.db.FindOne(ctx, bson.M{"title": title})
	if one.Err() != nil && !errors.Is(one.Err(), mongo.ErrNoDocuments) {
		br.logger.Errorf("error find book with ID: %v", one.Err())
		return nil, one.Err()
	}
	if one.Err() != nil && errors.Is(one.Err(), mongo.ErrNoDocuments) {
		br.logger.Warnf("book with this title not found: %s", title)
		return nil, errs.ErrBookDoesNotExists
	}

	var book repomodels.BookModel
	if err := one.Decode(&book); err != nil {
		br.logger.Errorf("error decoding book: %v", err)
		return nil, err
	}

	br.logger.Infof("found book with title: %s", title)

	return br.convertToBookModel(&book), nil
}

func (br *BookRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	br.logger.Infof("deleting book with ID: %s", ID)

	one, err := br.db.DeleteOne(ctx, bson.M{"_id": ID})
	if err != nil {
		br.logger.Errorf("error deleting book: %v", err)
		return err
	}

	if one.DeletedCount == 0 {
		br.logger.Warnf("book with this ID not found %s", ID)
		return errs.ErrBookDoesNotExists
	}

	br.logger.Infof("deleted book with ID: %s", ID)

	return nil
}

func (br *BookRepo) Update(ctx context.Context, book *models.BookModel) error {
	br.logger.Infof("updating book with ID: %s", book.ID)

	updateData := bson.M{
		"$set": bson.M{
			"title":           book.Title,
			"author":          book.Author,
			"publisher":       book.Publisher,
			"copies_number":   book.CopiesNumber,
			"rarity":          book.Rarity,
			"genre":           book.Genre,
			"publishing_year": book.PublishingYear,
			"language":        book.Language,
			"age_limit":       book.AgeLimit,
		},
	}

	one, err := br.db.UpdateOne(ctx, bson.M{"_id": book.ID}, updateData)
	if err != nil {
		br.logger.Errorf("error updating book: %v", err)
		return err
	}

	if one.MatchedCount == 0 {
		br.logger.Warnf("book with this ID not found %s", book.ID)
		return errs.ErrBookDoesNotExists
	}

	br.logger.Infof("updated book with ID: %s", book.ID)

	return nil
}

func (br *BookRepo) GetByParams(ctx context.Context, params *dto.BookParamsDTO) ([]*models.BookModel, error) {
	br.logger.Printf("selecting books with params")

	filter := br.getFilterByParams(params)

	findOptions := options.Find()
	findOptions.SetLimit(int64(params.Limit))
	findOptions.SetSkip(int64(params.Offset))

	cursor, err := br.db.Find(ctx, filter, findOptions)
	if err != nil {
		br.logger.Printf("error selecting books with params: %v", err)
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err = cursor.Close(ctx)
		if err != nil {
			fmt.Println("error close cursor")
		}
	}(cursor, ctx)

	var coreBooks []*repomodels.BookModel
	if err = cursor.All(ctx, &coreBooks); err != nil {
		br.logger.Printf("error decoding books: %v", err)
		return nil, err
	}

	if len(coreBooks) == 0 {
		br.logger.Printf("books not found with these params")
		return nil, errs.ErrBookDoesNotExists
	}

	br.logger.Printf("found %d books", len(coreBooks))

	books := make([]*models.BookModel, len(coreBooks))
	for i, book := range coreBooks {
		books[i] = br.convertToBookModel(book)
	}

	return books, nil
}

func (br *BookRepo) getFilterByParams(params *dto.BookParamsDTO) bson.M {
	filter := bson.M{}

	if params.Title != "" {
		filter["title"] = bson.M{"$regex": params.Title, "$options": "i"}
	}
	if params.Author != "" {
		filter["author"] = bson.M{"$regex": params.Author, "$options": "i"}
	}
	if params.Publisher != "" {
		filter["publisher"] = bson.M{"$regex": params.Publisher, "$options": "i"}
	}
	if params.CopiesNumber != 0 {
		filter["copies_number"] = params.CopiesNumber
	}
	if params.Rarity != "" {
		filter["rarity"] = params.Rarity
	}
	if params.Genre != "" {
		filter["genre"] = bson.M{"$regex": params.Genre, "$options": "i"}
	}
	if params.PublishingYear != 0 {
		filter["publishing_year"] = params.PublishingYear
	}
	if params.Language != "" {
		filter["language"] = bson.M{"$regex": params.Language, "$options": "i"}
	}
	if params.AgeLimit != 0 {
		filter["age_limit"] = params.AgeLimit
	}

	return filter
}

func (br *BookRepo) convertToBookModel(book *repomodels.BookModel) *models.BookModel {
	return &models.BookModel{
		ID:             book.ID,
		Title:          book.Title,
		Author:         book.Author,
		Publisher:      book.Publisher,
		CopiesNumber:   book.CopiesNumber,
		Rarity:         book.Rarity,
		Genre:          book.Genre,
		PublishingYear: book.PublishingYear,
		Language:       book.Language,
		AgeLimit:       book.AgeLimit,
	}
}

func (br *BookRepo) convertToRepoBookModel(book *models.BookModel) *repomodels.BookModel {
	return &repomodels.BookModel{
		ID:             book.ID,
		Title:          book.Title,
		Author:         book.Author,
		Publisher:      book.Publisher,
		CopiesNumber:   book.CopiesNumber,
		Rarity:         book.Rarity,
		Genre:          book.Genre,
		PublishingYear: book.PublishingYear,
		Language:       book.Language,
		AgeLimit:       book.AgeLimit,
	}
}
