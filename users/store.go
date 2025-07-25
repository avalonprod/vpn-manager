package users

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

const usersCollection = "users"

type store struct {
	db *mongo.Collection
}

func NewStore(db *mongo.Database) *store {
	return &store{
		db: db.Collection(usersCollection),
	}
}

func (s *store) Create(ctx context.Context, user User) error {
	_, err := s.db.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrUserAlreadyExists
		}
		return err
	}

	return err
}

func (s *store) GetByID(ctx context.Context, ID int64) (User, error) {
	var user User
	err := s.db.FindOne(ctx, map[string]interface{}{"_id": ID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	return user, nil
}

func (s *store) GetAll(ctx context.Context) ([]User, error) {
	cursor, err := s.db.Find(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []User
	for cursor.Next(ctx) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (s *store) CountUsers(ctx context.Context) (int64, error) {
	count, err := s.db.CountDocuments(ctx, map[string]interface{}{})
	if err != nil {
		return 0, err
	}

	return count, nil
}
