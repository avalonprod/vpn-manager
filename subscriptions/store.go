package subscriptions

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const subscriptionsCollection = "subscriptions"

type store struct {
	db *mongo.Collection
}

func NewStore(db *mongo.Database) *store {
	return &store{
		db: db.Collection(subscriptionsCollection),
	}
}

func (s *store) Create(ctx context.Context, subscription Subscription) error {
	_, err := s.db.InsertOne(ctx, subscription)
	return err
}

func (s *store) GetByUserID(ctx context.Context, userID int64) (*Subscription, error) {
	filter := bson.M{
		"user_id": userID,
	}

	var subscription Subscription

	err := s.db.FindOne(ctx, filter).Decode(&subscription)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}

	return &subscription, nil
}

func (s *store) GetExpiredSubscriptions(ctx context.Context) ([]Subscription, error) {
	filter := bson.M{
		"expires_at": bson.M{"$lt": time.Now().UTC()},
		"active":     true,
	}

	cursor, err := s.db.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var subscriptions []Subscription
	if err := cursor.All(ctx, &subscriptions); err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func (s *store) DeactivateExpiredSubscriptions(ctx context.Context) error {
	filter := bson.M{
		"expires_at": bson.M{"$lt": time.Now()},
		"active":     true,
	}
	update := bson.M{
		"$set": bson.M{"active": false},
	}

	_, err := s.db.UpdateMany(ctx, filter, update)

	return err
}

func (s *store) Update(ctx context.Context, userID int64, ID string, input Subscription) error {
	ObjectID, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":     ObjectID,
		"user_id": userID,
	}
	update := bson.M{
		"$set": bson.M{
			"plan_id":    input.PlanID,
			"active":     input.Active,
			"expires_at": input.ExpiresAt,
		},
	}

	res, err := s.db.UpdateOne(ctx, filter, update)
	if res.ModifiedCount == 0 {
		return ErrSubscriptionNotFound
	}

	return err
}

func (s *store) DeactivateSubscription(ctx context.Context, userID int64, ID string) error {
	ObjectID, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":     ObjectID,
		"user_id": userID,
	}
	update := bson.M{
		"$set": bson.M{"active": false},
	}

	res, err := s.db.UpdateOne(ctx, filter, update)
	if res.ModifiedCount == 0 {
		return ErrSubscriptionNotFound
	}

	return err
}
