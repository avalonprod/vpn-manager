package admin

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const auditCollection = "admin_audit"

type AuditEntry struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	Actor     string    `bson:"actor" json:"actor"`
	Action    string    `bson:"action" json:"action"`
	Target    string    `bson:"target,omitempty" json:"target,omitempty"`
	Details   string    `bson:"details,omitempty" json:"details,omitempty"`
	IP        string    `bson:"ip,omitempty" json:"ip,omitempty"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type AuditStore struct {
	db *mongo.Collection
}

func NewAuditStore(db *mongo.Database) *AuditStore {
	return &AuditStore{db: db.Collection(auditCollection)}
}

func (s *AuditStore) Write(ctx context.Context, entry AuditEntry) error {
	entry.CreatedAt = time.Now().UTC()

	_, err := s.db.InsertOne(ctx, entry)

	return err
}

func (s *AuditStore) List(ctx context.Context, limit, offset int) ([]AuditEntry, int64, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))

	cursor, err := s.db.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	entries := make([]AuditEntry, 0, limit)
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, 0, err
	}

	total, err := s.db.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

func (h *Handler) audit(ctx context.Context, actor, ip, action, target, details string) {
	if h.auditStore == nil {
		return
	}

	entry := AuditEntry{
		Actor:   actor,
		Action:  action,
		Target:  target,
		Details: details,
		IP:      ip,
	}

	if err := h.auditStore.Write(ctx, entry); err != nil {
		h.logger.Errorf("admin: failed to write audit entry %s: %v", action, err)
	}
}
