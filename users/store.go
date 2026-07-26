package users

import (
	"context"
	"regexp"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	err := s.db.FindOne(ctx, bson.M{"_id": ID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	return user, nil
}

func (s *store) GetAll(ctx context.Context) ([]User, error) {
	cursor, err := s.db.Find(ctx, bson.M{})
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
	count, err := s.db.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, err
	}

	return count, nil
}

// buildListFilter превращает ListFilter в bson-запрос. Пользовательский ввод
// экранируется, чтобы строка поиска не могла стать regex-инъекцией.
func buildListFilter(f ListFilter) bson.M {
	filter := bson.M{}

	switch f.Blocked {
	case BlockedOnly:
		filter["is_blocked"] = true
	case BlockedExclude:
		filter["is_blocked"] = bson.M{"$ne": true}
	}

	if f.Search != "" {
		safe := regexp.QuoteMeta(f.Search)
		or := []bson.M{
			{"username": bson.M{"$regex": safe, "$options": "i"}},
			{"first_name": bson.M{"$regex": safe, "$options": "i"}},
		}

		if id, err := strconv.ParseInt(f.Search, 10, 64); err == nil {
			or = append(or, bson.M{"_id": id})
		}

		filter["$or"] = or
	}

	return filter
}

var allowedSortFields = map[string]string{
	"created_at":  "created_at",
	"last_active": "last_active",
	"id":          "_id",
}

func (s *store) List(ctx context.Context, f ListFilter) ([]User, error) {
	sortField, ok := allowedSortFields[f.SortField]
	if !ok {
		sortField = "created_at"
	}

	direction := -1
	if f.SortAsc {
		direction = 1
	}

	opts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: direction}, {Key: "_id", Value: direction}}).
		SetSkip(int64(f.Offset)).
		SetLimit(int64(f.Limit))

	cursor, err := s.db.Find(ctx, buildListFilter(f), opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	users := make([]User, 0, f.Limit)
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}

func (s *store) Count(ctx context.Context, f ListFilter) (int64, error) {
	return s.db.CountDocuments(ctx, buildListFilter(f))
}

func (s *store) CountBlocked(ctx context.Context) (int64, error) {
	return s.db.CountDocuments(ctx, bson.M{"is_blocked": true})
}

func (s *store) CountCreatedSince(ctx context.Context, since time.Time) (int64, error) {
	return s.db.CountDocuments(ctx, bson.M{"created_at": bson.M{"$gte": since}})
}

func (s *store) CountActiveSince(ctx context.Context, since time.Time) (int64, error) {
	return s.db.CountDocuments(ctx, bson.M{"last_active": bson.M{"$gte": since}})
}

func (s *store) SetBlocked(ctx context.Context, userID int64, blocked bool, reason string) error {
	update := bson.M{"$set": bson.M{"is_blocked": blocked}}
	if blocked {
		update["$set"].(bson.M)["block_reason"] = reason
		update["$set"].(bson.M)["blocked_at"] = time.Now().UTC()
	} else {
		update["$unset"] = bson.M{"block_reason": "", "blocked_at": ""}
	}

	res, err := s.db.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (s *store) TouchLastActive(ctx context.Context, userID int64) error {
	_, err := s.db.UpdateOne(ctx, bson.M{"_id": userID},
		bson.M{"$set": bson.M{"last_active": time.Now().UTC()}})

	return err
}

// SignupsByDay возвращает число регистраций по дням в UTC начиная с since.
func (s *store) SignupsByDay(ctx context.Context, since time.Time) ([]DailyCount, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"created_at": bson.M{"$gte": since}}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{"$dateToString": bson.M{
				"format":   "%Y-%m-%d",
				"date":     "$created_at",
				"timezone": "UTC",
			}},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}

	cursor, err := s.db.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make([]DailyCount, 0)
	if err := cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetManyByIDs отдаёт пользователей по списку ID одним запросом.
func (s *store) GetManyByIDs(ctx context.Context, ids []int64) (map[int64]User, error) {
	if len(ids) == 0 {
		return map[int64]User{}, nil
	}

	cursor, err := s.db.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	users := make(map[int64]User, len(ids))
	for cursor.Next(ctx) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users[user.ID] = user
	}

	return users, cursor.Err()
}
