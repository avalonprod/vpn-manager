package peers

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const peersCollection = "peers"

type store struct {
	db *mongo.Collection
}

func NewStore(db *mongo.Database) *store {
	return &store{
		db: db.Collection(peersCollection),
	}
}

func (s *store) Create(ctx context.Context, peer Peer) (string, error) {
	res, err := s.db.InsertOne(ctx, peer)
	if err != nil {
		return "", err
	}

	id := res.InsertedID.(primitive.ObjectID).Hex()

	return id, nil
}

func (s *store) GetByID(ctx context.Context, ID string) (Peer, error) {
	var peer Peer

	ObjectID, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		return Peer{}, err
	}

	filter := map[string]interface{}{"_id": ObjectID}

	err = s.db.FindOne(ctx, filter).Decode(&peer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Peer{}, ErrPeerNotFound
		}
		return Peer{}, err
	}

	return peer, nil
}

func (s *store) DeletePeersByUserID(ctx context.Context, userID int64) error {
	filter := bson.M{"user_id": userID}

	_, err := s.db.DeleteOne(ctx, filter)

	return err
}

func (s *store) GetPeersByUserID(ctx context.Context, userID int64) ([]Peer, error) {
	var peers []Peer

	filter := bson.M{"user_id": userID}

	cursor, err := s.db.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var peer Peer
		if err := cursor.Decode(&peer); err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return peers, nil
}
