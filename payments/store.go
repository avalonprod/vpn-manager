package payments

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const invoicesCollection = "invoices"

type store struct {
	db *mongo.Collection
}

func NewStore(db *mongo.Database) *store {
	return &store{
		db: db.Collection(invoicesCollection),
	}
}

func (s *store) Create(ctx context.Context, invoice Invoice) (string, error) {
	res, err := s.db.InsertOne(ctx, invoice)
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("inserted ID is not an ObjectID: %T", res.InsertedID)
	}

	return oid.Hex(), err
}

func (s *store) GetByID(ctx context.Context, userID int64, ID string) (Invoice, error) {
	var invoice Invoice

	ObjectID, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		return Invoice{}, err
	}

	filter := map[string]interface{}{"_id": ObjectID, "user_id": userID}

	err = s.db.FindOne(ctx, filter).Decode(&invoice)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Invoice{}, ErrInvoiceNotFound
		}
		return Invoice{}, err
	}

	return invoice, nil
}

func (s *store) SetStatus(ctx context.Context, userID int64, ID, status string) error {
	ObjectID, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		return err
	}
	filter := bson.M{"_id": ObjectID, "user_id": userID}
	update := bson.M{"$set": bson.M{"status": status}}

	_, err = s.db.UpdateOne(ctx, filter, update)
	return err
}

func (s *store) GetAllCompletedInvoices(ctx context.Context) ([]Invoice, error) {
	filter := bson.M{"status": StatusCompleted}
	cursor, err := s.db.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invoices []Invoice
	for cursor.Next(ctx) {
		var invoice Invoice
		if err := cursor.Decode(&invoice); err != nil {
			return nil, err
		}
		invoices = append(invoices, invoice)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return invoices, nil
}

func (s *store) CountCompletedInvoices(ctx context.Context) (int64, error) {
	filter := bson.M{"status": StatusCompleted}
	count, err := s.db.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func buildListFilter(f ListFilter) bson.M {
	filter := bson.M{}

	if f.Status != "" {
		filter["status"] = f.Status
	}
	if f.UserID != 0 {
		filter["user_id"] = f.UserID
	}

	return filter
}

func (s *store) List(ctx context.Context, f ListFilter) ([]Invoice, int64, error) {
	filter := buildListFilter(f)

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(f.Offset)).
		SetLimit(int64(f.Limit))

	cursor, err := s.db.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	invoices := make([]Invoice, 0, f.Limit)
	if err := cursor.All(ctx, &invoices); err != nil {
		return nil, 0, err
	}

	total, err := s.db.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

func (s *store) GetByUserID(ctx context.Context, userID int64, limit int) ([]Invoice, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := s.db.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	invoices := make([]Invoice, 0, limit)
	if err := cursor.All(ctx, &invoices); err != nil {
		return nil, err
	}

	return invoices, nil
}

func (s *store) CountByStatus(ctx context.Context, status string) (int64, error) {
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}

	return s.db.CountDocuments(ctx, filter)
}

// planLookup подтягивает цену тарифа к счёту: invoices.plan_id — строка,
// как и plans._id, поэтому $lookup работает напрямую.
var planLookup = bson.D{{Key: "$lookup", Value: bson.M{
	"from":         "plans",
	"localField":   "plan_id",
	"foreignField": "_id",
	"as":           "plan",
}}}

var planPrice = bson.M{"$ifNull": bson.A{bson.M{"$arrayElemAt": bson.A{"$plan.price", 0}}, 0}}

// RevenueSince считает суммарную выручку по оплаченным счетам с момента since.
// Нулевое время означает «за всё время».
func (s *store) RevenueSince(ctx context.Context, since time.Time) (float64, error) {
	match := bson.M{"status": StatusCompleted}
	if !since.IsZero() {
		match["created_at"] = bson.M{"$gte": since}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		planLookup,
		{{Key: "$group", Value: bson.M{
			"_id":     nil,
			"revenue": bson.M{"$sum": planPrice},
		}}},
	}

	cursor, err := s.db.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var result []struct {
		Revenue float64 `bson:"revenue"`
	}
	if err := cursor.All(ctx, &result); err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, nil
	}

	return result[0].Revenue, nil
}

// RevenueByDay строит временной ряд выручки по дням в UTC начиная с since.
func (s *store) RevenueByDay(ctx context.Context, since time.Time) ([]DailyRevenue, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"status":     StatusCompleted,
			"created_at": bson.M{"$gte": since},
		}}},
		planLookup,
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{"$dateToString": bson.M{
				"format":   "%Y-%m-%d",
				"date":     "$created_at",
				"timezone": "UTC",
			}},
			"revenue": bson.M{"$sum": planPrice},
			"count":   bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}

	cursor, err := s.db.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make([]DailyRevenue, 0)
	if err := cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// RevenueByPlan разбивает выручку по тарифам за период с since.
func (s *store) RevenueByPlan(ctx context.Context, since time.Time) ([]PlanRevenue, error) {
	match := bson.M{"status": StatusCompleted}
	if !since.IsZero() {
		match["created_at"] = bson.M{"$gte": since}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		planLookup,
		{{Key: "$group", Value: bson.M{
			"_id":     "$plan_id",
			"revenue": bson.M{"$sum": planPrice},
			"count":   bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"revenue": -1}}},
	}

	cursor, err := s.db.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make([]PlanRevenue, 0)
	if err := cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}
