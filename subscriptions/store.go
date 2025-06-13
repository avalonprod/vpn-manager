package subscriptions

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
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

func (s *store) HasTrialSubscription(ctx context.Context, userID int64) (bool, error) {
	filter := bson.M{
		"user_id": userID,
		"plan":    PlanTrial,
	}

	count, err := s.db.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to count subscriptions: %w", err)
	}

	return count > 0, nil
}
