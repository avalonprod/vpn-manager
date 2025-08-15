package peers

import (
	"context"
	"fmt"
	"time"

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
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("inserted ID is not an ObjectID: %T", res.InsertedID)
	}

	return oid.Hex(), err
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

func (s *store) GetPeerByUserID(ctx context.Context, userID int64) (Peer, error) {
	var peer Peer

	filter := map[string]interface{}{"user_id": userID}

	err := s.db.FindOne(ctx, filter).Decode(&peer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Peer{}, ErrPeerNotFound
		}
		return Peer{}, err
	}

	return peer, nil
}

func (s *store) GetActivePeerByUserID(ctx context.Context, userID int64) (Peer, error) {
	var peer Peer

	filter := map[string]interface{}{"user_id": userID, "is_active": true}

	err := s.db.FindOne(ctx, filter).Decode(&peer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Peer{}, ErrPeerNotFound
		}
		return Peer{}, err
	}

	return peer, nil
}

func (s *store) UpdateSubs(ctx context.Context, id string, subs []Sub) error {
	ObjectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": ObjectID}
	update := bson.M{"$set": bson.M{"subs": subs}}

	_, err = s.db.UpdateOne(ctx, filter, update)
	return err
}

func (s *store) SetActive(ctx context.Context, userID int64) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{"is_active": true}}

	_, err := s.db.UpdateOne(ctx, filter, update)
	return err
}

func (s *store) Deactivate(ctx context.Context, userID int64) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{"is_active": false}}

	_, err := s.db.UpdateOne(ctx, filter, update)
	return err
}

func (s *store) SetImported(ctx context.Context, userID int64, val bool, importedAt time.Time) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{"is_imported": val, "imported_at": importedAt}}

	_, err := s.db.UpdateOne(ctx, filter, update)
	return err
}

func (s *store) GetPeersForUsers(ctx context.Context, userIDs []int64) (map[int64]Peer, error) {
	if len(userIDs) == 0 {
		return map[int64]Peer{}, nil
	}

	cur, err := s.db.Find(ctx, bson.M{
		"user_id": bson.M{"$in": userIDs},
	})
	if err != nil {
		return nil, err
	}

	defer cur.Close(ctx)

	peers := make(map[int64]Peer)

	for cur.Next(ctx) {
		var p Peer
		if err := cur.Decode(&p); err != nil {
			return nil, err
		}

		peers[p.UserID] = p
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return peers, nil
}
