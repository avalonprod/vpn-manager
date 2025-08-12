package tasks

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const tasksCollection = "tasks"

type Store struct {
	db *mongo.Collection
}

func NewStore(db *mongo.Database) *Store {
	return &Store{
		db: db.Collection(tasksCollection)}
}

func (s *Store) Enqueue(ctx context.Context, t Task) error {
	t.Status = StatusPending
	t.CreatedAt = time.Now().UTC()
	t.UpdatedAt = t.CreatedAt
	_, err := s.db.InsertOne(ctx, t)
	return err
}

func (s *Store) ClaimDue(ctx context.Context) (*Task, error) {
	filter := bson.M{"status": StatusPending, "run_at": bson.M{"$lte": time.Now().UTC()}}
	update := bson.M{"$set": bson.M{"status": StatusRunning, "updated_at": time.Now().UTC()}, "$inc": bson.M{"attempts": 1}}
	opts := options.FindOneAndUpdate().SetSort(bson.M{"run_at": 1}).SetReturnDocument(options.After)
	var t Task
	if err := s.db.FindOneAndUpdate(ctx, filter, update, opts).Decode(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) Reschedule(ctx context.Context, ID string, runAt time.Time) error {
	ObjectID, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		return err
	}

	filter := map[string]interface{}{"_id": ObjectID}

	_, err = s.db.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{"status": StatusPending, "run_at": runAt, "updated_at": time.Now().UTC()},
	})

	return err
}

func (s *Store) SetStatus(ctx context.Context, ID string, status Status) error {
	ObjectID, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		return err
	}

	filter := map[string]interface{}{"_id": ObjectID}

	_, err = s.db.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"status": status, "updated_at": time.Now().UTC()}})

	return err
}
