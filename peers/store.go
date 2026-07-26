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
	if err != nil {
		return "", err
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("inserted ID is not an ObjectID: %T", res.InsertedID)
	}

	return oid.Hex(), nil
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

func (s *store) Totals(ctx context.Context) (Totals, error) {
	var totals Totals

	counts := []struct {
		target *int64
		filter bson.M
	}{
		{&totals.Total, bson.M{}},
		{&totals.Active, bson.M{"is_active": true}},
		{&totals.Imported, bson.M{"is_imported": true}},
	}

	for _, c := range counts {
		count, err := s.db.CountDocuments(ctx, c.filter)
		if err != nil {
			return Totals{}, err
		}
		*c.target = count
	}

	return totals, nil
}

func (s *store) CountByLocation(ctx context.Context) ([]LocationCount, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"is_active": true}}},
		{{Key: "$unwind", Value: "$subs"}},
		{{Key: "$match", Value: bson.M{"subs.enabled": true}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"server_id": "$subs.server_id", "location": "$subs.location"},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":       0,
			"server_id": "$_id.server_id",
			"location":  "$_id.location",
			"count":     1,
		}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
	}

	cursor, err := s.db.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make([]LocationCount, 0)
	if err := cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
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
