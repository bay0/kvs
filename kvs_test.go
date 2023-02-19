package kvs

import (
	"fmt"
	"testing"
)

type IntValue int

func (iv IntValue) Clone() Value {
	return iv
}

type Person struct {
	Name string
	Age  int
}

func (p Person) Clone() Value {
	return Person{
		Name: p.Name,
		Age:  p.Age,
	}
}

func TestSet(t *testing.T) {
	store := NewKeyValueStore(10)
	value := &Person{
		Name: "Alice",
		Age:  30,
	}

	err := store.Set("person", value)
	if err != nil {
		t.Errorf("Set returned an error: %v", err)
	}

	val, err := store.Get("person")
	if err != nil {
		t.Errorf("Get returned an error: %v", err)
	}
	if val == nil {
		t.Error("Get returned nil value")
	}
}

func TestGet(t *testing.T) {
	store := NewKeyValueStore(10)
	value := &Person{
		Name: "Alice",
		Age:  30,
	}

	if err := store.Set("person", value); err != nil {
		t.Errorf("Set returned an error: %v", err)
	}

	val, err := store.Get("person")
	if err != nil {
		t.Errorf("Get returned an error: %v", err)
	}
	if val == nil {
		t.Error("Get returned nil value")
	}
}

func TestDelete(t *testing.T) {
	store := NewKeyValueStore(10)
	value := &Person{
		Name: "Alice",
		Age:  30,
	}

	if err := store.Set("person", value); err != nil {
		t.Errorf("Set returned an error: %v", err)
	}

	err := store.Delete("person")
	if err != nil {
		t.Errorf("Delete returned an error: %v", err)
	}

	val, err := store.Get("person")
	if err == nil {
		t.Errorf("Get did not return an error for deleted key")
	}
	if val != nil {
		t.Error("Get returned non-nil value for deleted key")
	}
}

func TestKeys(t *testing.T) {
	store := NewKeyValueStore(10)
	value := &Person{
		Name: "Alice",
		Age:  30,
	}

	if err := store.Set("person", value); err != nil {
		t.Errorf("Set returned an error: %v", err)
	}

	keys, err := store.Keys()

	if err != nil {
		t.Errorf("Keys returned an error: %v", err)
	}

	if len(keys) != 1 || keys[0] != "person" {
		t.Errorf("Keys returned unexpected result: %v", keys)
	}
}

func TestKeyValueStore(t *testing.T) {
	t.Run("Set", TestSet)
	t.Run("Get", TestGet)
	t.Run("Delete", TestDelete)
	t.Run("Keys", TestKeys)
}

func TestKeyValueStore_Concurrent(t *testing.T) {
	store := NewKeyValueStore(10)

	// Set up a channel to communicate between goroutines
	done := make(chan bool)

	// Use multiple goroutines to write to the key-value store
	for i := 0; i < 10; i++ {
		go func(j int) {
			for k := 0; k < 1000; k++ {
				key := fmt.Sprintf("key-%d-%d", j, k)
				err := store.Set(key, IntValue(j))
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish writing to the key-value store
	for i := 0; i < 10; i++ {
		<-done
	}

	// Use multiple goroutines to read from the key-value store
	for i := 0; i < 10; i++ {
		go func(j int) {
			keys, err := store.Keys()
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			for k := 0; k < 1000 && k < len(keys); k++ {
				key := fmt.Sprintf("key-%d-%d", j, k)
				val, err := store.Get(key)
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if iv, ok := val.(IntValue); !ok || iv != IntValue(j) {
					t.Errorf("Expected IntValue(%d), got %v", j, val)
				}
			}
			done <- true
		}(i)
	}
}

func TestKeyValueStore_Struct(t *testing.T) {
	store := NewKeyValueStore(5)

	// Add some people to the store
	if err := store.Set("john", Person{Name: "John Doe", Age: 42}); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := store.Set("jane", Person{Name: "Jane Doe", Age: 36}); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := store.Set("bob", Person{Name: "Bob Smith", Age: 27}); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Retrieve a person from the store
	if val, err := store.Get("john"); err != nil {
		t.Errorf("Expected no error, got %v", err)
	} else if p, ok := val.(Person); !ok {
		t.Errorf("Expected a Person value, got %v", val)
	} else if p.Name != "John Doe" || p.Age != 42 {
		t.Errorf("Expected Person{Name: 'John Doe', Age: 42}, got %v", p)
	}

	// Update a person in the store
	if err := store.Set("john", Person{Name: "John Smith", Age: 43}); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Delete a person from the store
	if err := store.Delete("bob"); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that the correct people are in the store
	expected := []Person{
		{Name: "John Smith", Age: 43},
		{Name: "Jane Doe", Age: 36},
	}
	var actual []Person
	keys, err := store.Keys()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	for _, key := range keys {
		if val, err := store.Get(key); err != nil {
			t.Errorf("Expected no error, got %v", err)
		} else if p, ok := val.(Person); !ok {
			t.Errorf("Expected a Person value, got %v", val)
		} else {
			actual = append(actual, p)
		}
	}

	if len(actual) != len(expected) {
		t.Errorf("Expected %d people, got %d", len(expected), len(actual))
	}
	for _, p := range expected {
		if !contains(actual, p) {
			t.Errorf("Expected %v, got %v", p, actual)
		}
	}
}

func contains(persons []Person, p Person) bool {
	for _, person := range persons {
		if person == p {
			return true
		}
	}
	return false
}

func BenchmarkSet(b *testing.B) {
	store := NewKeyValueStore(10)
	value := &Person{
		Name: "Alice",
		Age:  30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := store.Set("person", value); err != nil {
			b.Errorf("Expected no error, got %v", err)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	store := NewKeyValueStore(10)
	value := &Person{
		Name: "Alice",
		Age:  30,
	}
	if err := store.Set("person", value); err != nil {
		b.Errorf("Expected no error, got %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.Get("person"); err != nil {
			b.Errorf("Expected no error, got %v", err)
		}
	}
}

func BenchmarkDelete(b *testing.B) {
	store := NewKeyValueStore(10)
	value := &Person{
		Name: "Alice",
		Age:  30,
	}
	if err := store.Set("person", value); err != nil {
		b.Errorf("Expected no error, got %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Get("person")
		if err == nil {
			if err := store.Delete("person"); err != nil {
				b.Errorf("Expected no error, got %v", err)
			}
		}
	}
}
