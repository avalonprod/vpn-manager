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
		"expires_at": bson.M{"$lt": time.Now().UTC()},
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

func (s *store) CancelSubscription(ctx context.Context, userID int64) error {
	filter := bson.M{
		"user_id": userID,
	}
	update := bson.M{
		"$set": bson.M{"auto_renewal": false},
	}

	res, err := s.db.UpdateOne(ctx, filter, update)
	if res.ModifiedCount == 0 {
		return ErrSubscriptionNotFound
	}

	return err
}

func (s *store) GetAllTrialSubscriptions(ctx context.Context) ([]Subscription, error) {
	cursor, err := s.db.Find(ctx, bson.M{
		"is_trial": true,
	})
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

func (s *store) GetExpiredTrialSubscriptions(ctx context.Context) ([]Subscription, error) {
	filter := bson.M{
		"is_trial":   true,
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

func (s *store) GetSubscriptionsForUsers(ctx context.Context, userIDs []int64) (map[int64]Subscription, error) {
	if len(userIDs) == 0 {
		return map[int64]Subscription{}, nil
	}

	cur, err := s.db.Find(ctx, bson.M{
		"user_id": bson.M{"$in": userIDs},
	})
	if err != nil {
		return nil, err
	}

	defer cur.Close(ctx)

	subs := make(map[int64]Subscription)

	for cur.Next(ctx) {
		var s Subscription
		if err := cur.Decode(&s); err != nil {
			return nil, err
		}

		subs[s.UserID] = s
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}

func (s *store) CountTrialSubscriptions(ctx context.Context) (int64, error) {
	count, err := s.db.CountDocuments(ctx, bson.M{"is_trial": true})
	if err != nil {
		return 0, err
	}
	return count, nil
}
