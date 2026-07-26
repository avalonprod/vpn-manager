package subscriptions

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (s *store) Totals(ctx context.Context) (Totals, error) {
	var totals Totals

	counts := []struct {
		target *int64
		filter bson.M
	}{
		{&totals.Total, bson.M{}},
		{&totals.Active, bson.M{"active": true}},
		{&totals.ActiveTrial, bson.M{"active": true, "is_trial": true}},
		{&totals.ActivePaid, bson.M{"active": true, "is_trial": bson.M{"$ne": true}}},
		{&totals.AutoRenewal, bson.M{"active": true, "auto_renewal": true}},
		{&totals.ExpiringIn3d, bson.M{"active": true, "expires_at": bson.M{
			"$gte": time.Now().UTC(),
			"$lte": time.Now().UTC().Add(72 * time.Hour),
		}}},
	}

	for _, c := range counts {
		count, err := s.db.CountDocuments(ctx, c.filter)
		if err != nil {
			return Totals{}, err
		}
		*c.target = count
	}

	return totals, nil
}

// CountByPlan показывает распределение активных подписок по тарифам.
func (s *store) CountByPlan(ctx context.Context) ([]PlanCount, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"active": true}}},
		{{Key: "$group", Value: bson.M{"_id": "$plan_id", "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
	}

	cursor, err := s.db.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make([]PlanCount, 0)
	if err := cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// CreatedByDay возвращает число новых подписок по дням в UTC начиная с since.
func (s *store) CreatedByDay(ctx context.Context, since time.Time, trialOnly *bool) ([]DailyCount, error) {
	match := bson.M{"created_at": bson.M{"$gte": since}}
	if trialOnly != nil {
		if *trialOnly {
			match["is_trial"] = true
		} else {
			match["is_trial"] = bson.M{"$ne": true}
		}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{"$dateToString": bson.M{
				"format":   "%Y-%m-%d",
				"date":     "$created_at",
				"timezone": "UTC",
			}},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}

	cursor, err := s.db.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make([]DailyCount, 0)
	if err := cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// List отдаёт подписки постранично, свежие сверху.
func (s *store) List(ctx context.Context, activeOnly bool, limit, offset int) ([]Subscription, int64, error) {
	filter := bson.M{}
	if activeOnly {
		filter["active"] = true
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))

	cursor, err := s.db.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	subs := make([]Subscription, 0, limit)
	if err := cursor.All(ctx, &subs); err != nil {
		return nil, 0, err
	}

	total, err := s.db.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return subs, total, nil
}
