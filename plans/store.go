package plans

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const plansCollection = "plans"

type store struct {
	db *mongo.Collection
}

func NewStore(db *mongo.Database) *store {
	return &store{
		db: db.Collection(plansCollection),
	}
}

func (s *store) GetAll(ctx context.Context) ([]Plan, error) {
	return s.find(ctx, bson.M{"is_active": true})
}

func (s *store) GetAllIncludingInactive(ctx context.Context) ([]Plan, error) {
	return s.find(ctx, bson.M{})
}

func (s *store) find(ctx context.Context, filter bson.M) ([]Plan, error) {
	opts := options.Find().SetSort(bson.D{{Key: "order", Value: 1}})

	cursor, err := s.db.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	plans := make([]Plan, 0)
	if err := cursor.All(ctx, &plans); err != nil {
		return nil, err
	}

	return plans, nil
}

func (s *store) GetByID(ctx context.Context, ID string) (Plan, error) {
	var plan Plan

	err := s.db.FindOne(ctx, bson.M{"_id": ID}).Decode(&plan)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Plan{}, ErrPlanNotFound
		}
		return Plan{}, err
	}

	return plan, nil
}

func (s *store) Create(ctx context.Context, plan Plan) (string, error) {
	res, err := s.db.InsertOne(ctx, plan)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", ErrPlanAlreadyExists
		}
		return "", err
	}

	switch id := res.InsertedID.(type) {
	case string:
		return id, nil
	case primitive.ObjectID:
		return id.Hex(), nil
	default:
		return "", ErrInvalidPlanID
	}
}

func (s *store) Update(ctx context.Context, ID string, fields bson.M) error {
	if len(fields) == 0 {
		return nil
	}

	res, err := s.db.UpdateOne(ctx, bson.M{"_id": ID}, bson.M{"$set": fields})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return ErrPlanNotFound
	}

	return nil
}

func (s *store) Delete(ctx context.Context, ID string) error {
	res, err := s.db.DeleteOne(ctx, bson.M{"_id": ID})
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return ErrPlanNotFound
	}

	return nil
}
