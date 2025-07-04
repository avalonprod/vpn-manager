package plans

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
	var plans []Plan

	filter := bson.M{"is_active": true}

	cursor, err := s.db.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var plan Plan
		if err := cursor.Decode(&plan); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return plans, nil
}

func (s *store) GetByID(ctx context.Context, ID string) (Plan, error) {
	var plan Plan

	filter := bson.M{"_id": ID}

	err := s.db.FindOne(ctx, filter).Decode(&plan)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Plan{}, ErrPlanNotFound
		}
		return Plan{}, err
	}

	return plan, nil
}
