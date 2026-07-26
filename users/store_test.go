package users

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// Строка поиска приходит от администратора и попадает в $regex — спецсимволы
// обязаны экранироваться, иначе ввод становится regex-инъекцией.
func TestBuildListFilterEscapesSearchInput(t *testing.T) {
	filter := buildListFilter(ListFilter{Search: "a.*b("})

	or, ok := filter["$or"].([]bson.M)
	if !ok {
		t.Fatalf("$or has type %T, want []bson.M", filter["$or"])
	}

	if len(or) != 2 {
		t.Fatalf("$or has %d clauses, want 2 (username, first_name)", len(or))
	}

	for _, clause := range or {
		for field, condition := range clause {
			regex, ok := condition.(bson.M)
			if !ok {
				t.Fatalf("%s: condition has type %T, want bson.M", field, condition)
			}

			if got := regex["$regex"]; got != `a\.\*b\(` {
				t.Errorf("%s: $regex = %v, want the escaped literal", field, got)
			}
		}
	}
}

// Числовая строка должна дополнительно искать точное совпадение по ID.
func TestBuildListFilterMatchesNumericIDs(t *testing.T) {
	filter := buildListFilter(ListFilter{Search: "12345"})

	or, ok := filter["$or"].([]bson.M)
	if !ok {
		t.Fatalf("$or has type %T, want []bson.M", filter["$or"])
	}

	if len(or) != 3 {
		t.Fatalf("$or has %d clauses, want 3 (username, first_name, _id)", len(or))
	}

	if got := or[2]["_id"]; got != int64(12345) {
		t.Errorf("_id clause = %v (%T), want int64(12345)", got, got)
	}
}

func TestBuildListFilterAppliesBlockedFilter(t *testing.T) {
	if got := buildListFilter(ListFilter{Blocked: BlockedOnly})["is_blocked"]; got != true {
		t.Errorf("blocked filter = %v, want true", got)
	}

	active, ok := buildListFilter(ListFilter{Blocked: BlockedExclude})["is_blocked"].(bson.M)
	if !ok || active["$ne"] != true {
		t.Errorf("active filter = %v, want {$ne: true}", active)
	}

	if _, present := buildListFilter(ListFilter{})["is_blocked"]; present {
		t.Error("empty filter must not constrain is_blocked")
	}
}

// Поле сортировки приходит из query-параметра: неизвестное значение должно
// откатываться к безопасному умолчанию, а не уходить в запрос как есть.
func TestAllowedSortFieldsAreWhitelisted(t *testing.T) {
	if _, ok := allowedSortFields["created_at"]; !ok {
		t.Error("created_at must be an allowed sort field")
	}

	if _, ok := allowedSortFields["$where"]; ok {
		t.Error("arbitrary fields must not be allowed for sorting")
	}
}
