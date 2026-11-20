package mongo

import (
	"errors"

	"github.com/endale-dev/vigil/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
)

type operationError struct {
	cause error
}

func (operationError) Error() string {
	return "MongoDB operation failed"
}

func (err operationError) Unwrap() error {
	return err.cause
}

func idToObjectID(id domain.ID) (bson.ObjectID, error) {
	objectID, err := bson.ObjectIDFromHex(string(id))
	if err != nil {
		return bson.NilObjectID, domain.ErrNotFound
	}
	return objectID, nil
}

func objectIDToID(id bson.ObjectID) domain.ID {
	return domain.ID(id.Hex())
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, driver.ErrNoDocuments) {
		return domain.ErrNotFound
	}
	if driver.IsDuplicateKeyError(err) {
		var serverError driver.ServerError
		if errors.As(err, &serverError) {
			switch {
			case serverError.HasErrorMessage("users_email_unique"):
				return domain.ErrEmailTaken
			case serverError.HasErrorMessage("users_status_slug_unique"):
				return domain.ErrSlugTaken
			}
		}
	}
	return operationError{cause: err}
}
