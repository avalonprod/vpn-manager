package payments

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const invoicesCollection = "invoices"

type store struct {
	db *mongo.Collection
}

func NewStore(db *mongo.Database) *store {
	return &store{
		db: db.Collection(invoicesCollection),
	}
}

func (s *store) Create(ctx context.Context, invoice Invoice) (string, error) {
	res, err := s.db.InsertOne(ctx, invoice)
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("inserted ID is not an ObjectID: %T", res.InsertedID)
	}

	return oid.Hex(), err
}

func (s *store) GetByID(ctx context.Context, userID int64, ID string) (Invoice, error) {
	var invoice Invoice

	ObjectID, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		return Invoice{}, err
	}

	filter := map[string]interface{}{"_id": ObjectID, "user_id": userID}

	err = s.db.FindOne(ctx, filter).Decode(&invoice)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Invoice{}, ErrInvoiceNotFound
		}
		return Invoice{}, err
	}

	return invoice, nil
}

func (s *store) SetStatus(ctx context.Context, userID int64, ID, status string) error {
	ObjectID, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		return err
	}
	filter := bson.M{"_id": ObjectID, "user_id": userID}
	update := bson.M{"$set": bson.M{"status": status}}

	_, err = s.db.UpdateOne(ctx, filter, update)
	return err
}
