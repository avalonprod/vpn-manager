package servers

import (
	"context"
	"fmt"
	"log"
	"vpn-manager/pkg/secret"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const serversCollection = "servers"

type store struct {
	db     *mongo.Collection
	cipher *secret.Cipher
}

func NewStore(db *mongo.Database, cipher *secret.Cipher) *store {
	return &store{
		db:     db.Collection(serversCollection),
		cipher: cipher,
	}
}

func (s *store) decodeToken(server *Server) error {
	token, err := s.cipher.Decrypt(server.AuthToken)
	if err != nil {
		return fmt.Errorf("decrypt auth token of server %s: %w", server.ID, err)
	}

	server.AuthToken = token

	return nil
}

func (s *store) GetAllActiveServers(ctx context.Context) ([]Server, error) {
	return s.find(ctx, bson.M{"is_active": true})
}

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

	for i := range servers {
		if err := s.decodeToken(&servers[i]); err != nil {
			log.Printf("servers: %v", err)
			servers[i].AuthToken = ""
		}
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

	if err := s.decodeToken(&server); err != nil {
		return Server{}, err
	}

	return server, nil
}

func (s *store) Create(ctx context.Context, server Server) (string, error) {
	token, err := s.cipher.Encrypt(server.AuthToken)
	if err != nil {
		return "", fmt.Errorf("encrypt auth token: %w", err)
	}

	server.AuthToken = token

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

	if raw, ok := fields["auth_token"].(string); ok {
		token, err := s.cipher.Encrypt(raw)
		if err != nil {
			return fmt.Errorf("encrypt auth token: %w", err)
		}
		fields["auth_token"] = token
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
