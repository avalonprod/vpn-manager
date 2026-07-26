package servers

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const serversCollection = "servers"

type store struct {
	db *mongo.Collection
}

func NewStore(db *mongo.Database) *store {
	return &store{
		db: db.Collection(serversCollection),
	}
}

func (s *store) GetAllActiveServers(ctx context.Context) ([]Server, error) {
	return s.find(ctx, bson.M{"is_active": true})
}

// GetAll возвращает все серверы, включая выключенные, — для админ-панели.
func (s *store) GetAll(ctx context.Context) ([]Server, error) {
	return s.find(ctx, bson.M{})
}

func (s *store) find(ctx context.Context, filter bson.M) ([]Server, error) {
	opts := options.Find().SetSort(bson.D{{Key: "location", Value: 1}})

	cursor, err := s.db.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	servers := make([]Server, 0)
	if err := cursor.All(ctx, &servers); err != nil {
		return nil, err
	}

	return servers, nil
}

func (s *store) GetByID(ctx context.Context, serverID string) (Server, error) {
	var server Server

	ObjectID, err := primitive.ObjectIDFromHex(serverID)
	if err != nil {
		return Server{}, ErrServerNotFound
	}

	filter := bson.M{"_id": ObjectID}

	err = s.db.FindOne(ctx, filter).Decode(&server)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Server{}, ErrServerNotFound
		}
		return Server{}, err
	}

	return server, nil
}

func (s *store) Create(ctx context.Context, server Server) (string, error) {
	res, err := s.db.InsertOne(ctx, server)
	if err != nil {
		return "", err
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", ErrInvalidServerID
	}

	return oid.Hex(), nil
}

func (s *store) Update(ctx context.Context, serverID string, fields bson.M) error {
	if len(fields) == 0 {
		return nil
	}

	ObjectID, err := primitive.ObjectIDFromHex(serverID)
	if err != nil {
		return ErrServerNotFound
	}

	res, err := s.db.UpdateOne(ctx, bson.M{"_id": ObjectID}, bson.M{"$set": fields})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return ErrServerNotFound
	}

	return nil
}

func (s *store) Delete(ctx context.Context, serverID string) error {
	ObjectID, err := primitive.ObjectIDFromHex(serverID)
	if err != nil {
		return ErrServerNotFound
	}

	res, err := s.db.DeleteOne(ctx, bson.M{"_id": ObjectID})
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return ErrServerNotFound
	}

	return nil
}

func (s *store) Count(ctx context.Context) (total int64, active int64, err error) {
	if total, err = s.db.CountDocuments(ctx, bson.M{}); err != nil {
		return 0, 0, err
	}

	if active, err = s.db.CountDocuments(ctx, bson.M{"is_active": true}); err != nil {
		return 0, 0, err
	}

	return total, active, nil
}
