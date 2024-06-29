package ut

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

type User struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Age   int
	Email string
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to the database: %v", err)
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("Failed to migrate the database: %v", err)
	}

	users := []User{
		{Name: "Alice", Age: 25, Email: "alice@example.com"},
		{Name: "Bob", Age: 30, Email: "bob@example.com"},
		{Name: "Charlie", Age: 35, Email: "charlie@example.com"},
		{Name: "David", Age: 40, Email: "david@example.com"},
	}

	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("Failed to seed the database: %v", err)
	}

	return db
}

func TestBuildGormQuery(t *testing.T) {
	db := setupTestDB(t)

	tests := []struct {
		name   string
		query  *QueryFull
		expect int64
	}{
		{
			name: "Test with age greater than 30",
			query: &QueryFull{
				QueryParse: &QueryParse{
					And: []*Kov{
						{Key: "age", Op: "gt", Value: 30},
					},
				},
			},
			expect: 2,
		},
		{
			name: "Test with name equals 'Alice'",
			query: &QueryFull{
				QueryParse: &QueryParse{
					And: []*Kov{
						{Key: "name", Op: "eq", Value: "Alice"},
					},
				},
			},
			expect: 1,
		},
		{
			name: "Test with email regex",
			query: &QueryFull{
				QueryParse: &QueryParse{
					And: []*Kov{
						{Key: "email", Op: OpRegex, Value: ".*@example.com"},
					},
				},
			},
			expect: 4,
		},
		{
			name: "Test with pagination",
			query: &QueryFull{
				QueryParse: &QueryParse{},
				BaseQuery: &BaseQuery{
					BasePage: &BasePage{
						Page:     1,
						PageSize: 2,
					},
				},
			},
			expect: 2,
		},
		{
			name: "Test with sorting",
			query: &QueryFull{
				QueryParse: &QueryParse{},
				BaseQuery: &BaseQuery{
					BaseSort: &BaseSort{
						SortAsc: []string{"age"},
					},
				},
			},
			expect: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RunGormQuery[User](tt.query, db)
			if err != nil {
				t.Fatalf("Failed to build query: %v", err)
			}

			if tt.query.GetCount {
				if result.Count != tt.expect {
					t.Errorf("Expected count %d, got %d", tt.expect, result.Count)
				}
			} else {
				if int64(len(result.Datas)) != tt.expect {
					t.Errorf("Expected %d results, got %d", tt.expect, len(result.Datas))
				}
			}
		})
	}
}
