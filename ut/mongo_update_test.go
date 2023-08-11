package ut

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"testing"
)

type Address struct {
	City string `bson:"city"`
	Zip  string `bson:"zip"`
}

type User struct {
	ID      primitive.ObjectID `bson:"_id"`
	Name    string             `bson:"name"`
	Age     int                `bson:"age"`
	Address Address            `bson:"address"`
}

func TestStructToBsonM(t *testing.T) {
	converter := &StructConverter{}
	converter.AddCustomIsZero(new(primitive.ObjectID), func(v reflect.Value, field reflect.StructField) bool {
		return v.Interface() == primitive.NilObjectID
	})

	user := User{Name: "Alice", Age: 30, Address: Address{City: "New York", Zip: "10001"}}
	result, _ := converter.StructToBsonM(user)
	if result["_id"] != nil {
		t.Errorf("Expected _id to be nil, got %v", result["_id"])
	}
	if result["name"] != "Alice" {
		t.Errorf("Expected name to be Alice, got %v", result["name"])
	}
	if result["age"] != 30 {
		t.Errorf("Expected age to be 30, got %v", result["age"])
	}

	freeze := []string{"name", "address.city"}
	resultWithFreeze, _ := converter.StructToBsonM(user, freeze...)
	if resultWithFreeze["name"] != nil {
		t.Errorf("Expected name to be nil, got %v", resultWithFreeze["name"])
	}
	address, ok := resultWithFreeze["address"].(bson.M)
	if !ok || address["city"] != nil {
		t.Errorf("Expected address.city to be nil, got %v", address["city"])
	}
}

type InnerAddress struct {
	Street string `bson:"street"`
}

type NestedAddress struct {
	City   string       `bson:"city"`
	Street InnerAddress `bson:"street,inline"`
}

type NestedUser struct {
	Name    string        `bson:"name"`
	Age     int           `bson:"age"`
	Address NestedAddress `bson:"address"`
}

func TestStructToBsonMWithNestedStruct(t *testing.T) {
	converter := &StructConverter{}

	user := NestedUser{Name: "Alice", Age: 30, Address: NestedAddress{City: "New York", Street: InnerAddress{Street: "123 Main St"}}}
	result, _ := converter.StructToBsonM(user)
	if result["name"] != "Alice" || result["age"] != 30 {
		t.Errorf("Unexpected values for name or age")
	}
	address, ok := result["address"].(bson.M)
	if !ok || address["city"] != "New York" || address["street"] != "123 Main St" {
		t.Errorf("Unexpected values for address")
	}

	freeze := []string{"name", "address.street"}
	resultWithFreeze, _ := converter.StructToBsonM(user, freeze...)
	if resultWithFreeze["name"] != nil {
		t.Errorf("Expected name to be nil, got %v", resultWithFreeze["name"])
	}
	addressWithFreeze, ok := resultWithFreeze["address"].(bson.M)
	if !ok || addressWithFreeze["city"] != "New York" || addressWithFreeze["street"] != nil {
		t.Errorf("Unexpected values for address with freeze")
	}

	freezeAll := []string{"name", "address.city", "address.street"}
	resultWithFreezeAll, _ := converter.StructToBsonM(user, freezeAll...)
	if resultWithFreezeAll["name"] != nil || resultWithFreezeAll["address"] != nil {
		t.Errorf("Expected name and address to be nil, got %v and %v", resultWithFreezeAll["name"], resultWithFreezeAll["address"])
	}

	// 测试没有标签的纯内联结构体
	type PureInlinePerson struct {
		Person
		Email string `bson:"email"`
	}
	originalPerson := Person{Name: "Alice", Age: 25, Location: Location{City: "New York"}}
	pureInlinePerson := PureInlinePerson{Person: originalPerson, Email: "alice@example.com"}
	resultPureInline, _ := converter.StructToBsonM(pureInlinePerson)
	if resultPureInline["name"] != "Alice" || resultPureInline["age"] != 25 || resultPureInline["city"] != "New York" || resultPureInline["email"] != "alice@example.com" {
		t.Errorf("Unexpected values for pure inline person")
	}
}

type Location struct {
	City   string `bson:"city"`
	Street string `bson:"street"`
}

type Person struct {
	Name     string   `bson:"name"`
	Age      int      `bson:"age"`
	Location Location `bson:"location,inline"`
}

func TestDiffToBsonM(t *testing.T) {
	converter := new(StructConverter)

	// 测试基本字段的差异
	original := Person{Name: "Alice", Age: 25}
	current := Person{Name: "Alice", Age: 30}
	diff, err := converter.DiffToBsonM(original, current)
	if err != nil {
		t.Fatal(err)
	}
	if diff["age"] != 30 {
		t.Errorf("Expected age to be 30, got %v", diff["age"])
	}

	// 测试内联结构体的差异
	original = Person{Name: "Alice", Location: Location{City: "New York"}}
	current = Person{Name: "Alice", Location: Location{City: "Los Angeles"}}
	diff, err = converter.DiffToBsonM(original, current)
	if err != nil {
		t.Fatal(err)
	}
	if diff["city"] != "Los Angeles" {
		t.Errorf("Expected city to be Los Angeles, got %v", diff["city"])
	}

	// 测试冻结字段
	original = Person{Name: "Alice", Age: 25}
	current = Person{Name: "Bob", Age: 30}
	diff, err = converter.DiffToBsonM(original, current, "name")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := diff["name"]; ok {
		t.Errorf("Expected name to be frozen, but it was included in the diff")
	}

	// 测试内联结构体的冻结字段
	original = Person{Name: "Alice", Location: Location{City: "New York"}}
	current = Person{Name: "Alice", Location: Location{City: "Los Angeles"}}
	diff, err = converter.DiffToBsonM(original, current, "location.city")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := diff["city"]; ok {
		t.Errorf("Expected city to be frozen, but it was included in the diff")
	}

	// 测试多层内联结构体
	type ExtendedPerson struct {
		Person
		Email string `bson:"email"`
	}
	originalExtended := ExtendedPerson{Person: original, Email: "alice@example.com"}
	currentExtended := ExtendedPerson{Person: current, Email: "bob@example.com"}
	diff, err = converter.DiffToBsonM(originalExtended, currentExtended)
	if err != nil {
		t.Fatal(err)
	}
	if diff["email"] != "bob@example.com" {
		t.Errorf("Expected email to be bob@example.com, got %v", diff["email"])
	}
	if diff["city"] != "Los Angeles" {
		t.Errorf("Expected city to be Los Angeles, got %v", diff["city"])
	}
}
