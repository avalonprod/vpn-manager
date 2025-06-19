package servers

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
	var servers []Server
	filter := bson.M{"is_active": true}

	cursor, err := s.db.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var server Server
		if err := cursor.Decode(&server); err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return servers, nil
}

func (s *store) GetByID(ctx context.Context, serverID string) (Server, error) {
	var server Server

	ObjectID, err := primitive.ObjectIDFromHex(serverID)
	if err != nil {
		return Server{}, err
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
