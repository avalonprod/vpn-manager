package bot

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserStack struct {
	UserID  int64    `bson:"user_id"`
	Screens []string `bson:"screens"`
}

type StackStore struct {
	db *mongo.Collection
}

func NewStackScreens(db *mongo.Database) *StackStore {
	return &StackStore{
		db: db.Collection("userStack"),
	}
}

func (s *StackStore) Push(ctx context.Context, userID int64, screen string) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$push": bson.M{"screens": screen},
	}
	opts := options.Update().SetUpsert(true)

	_, err := s.db.UpdateOne(ctx, filter, update, opts)
	return err
}

func (s *StackStore) PopAndPeek(ctx context.Context, userID int64) (string, error) {
	var result UserStack

	err := s.db.FindOne(ctx, bson.M{"user_id": userID}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil
		}
		return "", err
	}

	if len(result.Screens) == 0 {
		return "", nil
	}

	newScreens := result.Screens[:len(result.Screens)-1]

	_, err = s.db.UpdateOne(ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"screens": newScreens}},
	)
	if err != nil {
		return "", err
	}

	return newScreens[len(newScreens)-1], nil
}
