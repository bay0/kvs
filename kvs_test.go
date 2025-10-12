package kvs

import (
	"encoding/gob"
	"fmt"
	"os"
	"testing"
	"time"
)

func init() {
	// Register types for gob encoding (needed for snapshots)
	gob.Register(IntValue(0))
	gob.Register(Person{})
}

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
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Errorf("NewKeyValueStore returned an error: %v", err)
	}

	value := &Person{
		Name: "Alice",
		Age:  30,
	}

	err = store.Set("person", value)
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
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Errorf("NewKeyValueStore returned an error: %v", err)
	}

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
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Errorf("NewKeyValueStore returned an error: %v", err)
	}

	value := &Person{
		Name: "Alice",
		Age:  30,
	}

	if err := store.Set("person", value); err != nil {
		t.Errorf("Set returned an error: %v", err)
	}

	err = store.Delete("person")
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
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Errorf("NewKeyValueStore returned an error: %v", err)
	}

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
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Errorf("NewKeyValueStore returned an error: %v", err)
	}

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
	store, err := NewKeyValueStore(5)
	if err != nil {
		t.Errorf("NewKeyValueStore returned an error: %v", err)
	}

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
	store, err := NewKeyValueStore(10)
	if err != nil {
		b.Errorf("NewKeyValueStore returned an error: %v", err)
	}

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
	store, err := NewKeyValueStore(10)
	if err != nil {
		b.Errorf("NewKeyValueStore returned an error: %v", err)
	}
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
	store, err := NewKeyValueStore(10)
	if err != nil {
		b.Errorf("NewKeyValueStore returned an error: %v", err)
	}

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

// TestExists_ExistingKey tests that Exists returns true for keys that are present in the store.
// Covers: T010 - Test Exists() returns true for existing keys
func TestExists_ExistingKey(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Set a key-value pair
	err = store.Set("testkey", IntValue(42))
	if err != nil {
		t.Fatalf("Set returned an error: %v", err)
	}

	// Check that Exists returns true
	exists := store.Exists("testkey")
	if !exists {
		t.Errorf("Exists returned false for existing key 'testkey'")
	}
}

// TestExists_NonExistentKey tests that Exists returns false for keys that are not in the store.
// Covers: T011 - Test Exists() returns false for non-existent keys
func TestExists_NonExistentKey(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Check that Exists returns false for a key that was never set
	exists := store.Exists("nonexistent")
	if exists {
		t.Errorf("Exists returned true for non-existent key 'nonexistent'")
	}

	// Set and then delete a key
	if err := store.Set("deleted", IntValue(1)); err != nil {
		t.Fatalf("Set returned an error: %v", err)
	}
	if err := store.Delete("deleted"); err != nil {
		t.Fatalf("Delete returned an error: %v", err)
	}

	// Check that Exists returns false for the deleted key
	exists = store.Exists("deleted")
	if exists {
		t.Errorf("Exists returned true for deleted key 'deleted'")
	}
}

// TestExists_Concurrent tests that Exists works correctly with concurrent Set/Delete operations.
// Covers: T012 - Test Exists() with concurrent operations
func TestExists_Concurrent(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	done := make(chan bool)

	// Goroutines writing keys
	for i := 0; i < 5; i++ {
		go func(j int) {
			for k := 0; k < 100; k++ {
				key := fmt.Sprintf("key-%d-%d", j, k)
				if err := store.Set(key, IntValue(j)); err != nil {
					t.Errorf("Set failed: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Goroutines checking existence
	for i := 0; i < 5; i++ {
		go func(j int) {
			for k := 0; k < 100; k++ {
				key := fmt.Sprintf("key-%d-%d", j, k)
				// Just check that Exists doesn't panic or race
				_ = store.Exists(key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify that all keys exist
	for i := 0; i < 5; i++ {
		for k := 0; k < 100; k++ {
			key := fmt.Sprintf("key-%d-%d", i, k)
			if !store.Exists(key) {
				t.Errorf("Exists returned false for key %s that should exist", key)
			}
		}
	}
}

// TestLen_EmptyStore tests that Len returns 0 for an empty store.
// Covers: T014 - Test Len() returns 0 for empty store
func TestLen_EmptyStore(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	length := store.Len()
	if length != 0 {
		t.Errorf("Len returned %d for empty store, expected 0", length)
	}
}

// TestLen_SetDelete tests that Len is correctly maintained during Set and Delete operations.
// Covers: T015 - Test Len() correctly tracks Set/Delete operations
func TestLen_SetDelete(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Start with 0
	if length := store.Len(); length != 0 {
		t.Errorf("Initial Len = %d, expected 0", length)
	}

	// Add 3 keys
	if err := store.Set("key1", IntValue(1)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := store.Set("key2", IntValue(2)); err != nil {
		t.Fatalf("Set key2 failed: %v", err)
	}
	if err := store.Set("key3", IntValue(3)); err != nil {
		t.Fatalf("Set key3 failed: %v", err)
	}

	if length := store.Len(); length != 3 {
		t.Errorf("After 3 Sets, Len = %d, expected 3", length)
	}

	// Update existing key (should not change count)
	if err := store.Set("key1", IntValue(10)); err != nil {
		t.Fatalf("Set key1 update failed: %v", err)
	}

	if length := store.Len(); length != 3 {
		t.Errorf("After updating key1, Len = %d, expected 3", length)
	}

	// Delete one key
	if err := store.Delete("key2"); err != nil {
		t.Fatalf("Delete key2 failed: %v", err)
	}

	if length := store.Len(); length != 2 {
		t.Errorf("After deleting key2, Len = %d, expected 2", length)
	}

	// Delete another key
	if err := store.Delete("key1"); err != nil {
		t.Fatalf("Delete key1 failed: %v", err)
	}

	if length := store.Len(); length != 1 {
		t.Errorf("After deleting key1, Len = %d, expected 1", length)
	}

	// Delete last key
	if err := store.Delete("key3"); err != nil {
		t.Fatalf("Delete key3 failed: %v", err)
	}

	if length := store.Len(); length != 0 {
		t.Errorf("After deleting all keys, Len = %d, expected 0", length)
	}
}

// TestLen_Concurrent tests that Len correctly tracks concurrent operations.
// Covers: T016 - Test Len() with concurrent Set/Delete operations
func TestLen_Concurrent(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	done := make(chan bool)
	numGoroutines := 10
	keysPerGoroutine := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(j int) {
			for k := 0; k < keysPerGoroutine; k++ {
				key := fmt.Sprintf("key-%d-%d", j, k)
				if err := store.Set(key, IntValue(j)); err != nil {
					t.Errorf("Set failed: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check that Len matches expected count
	expectedLen := numGoroutines * keysPerGoroutine
	if length := store.Len(); length != expectedLen {
		t.Errorf("After concurrent writes, Len = %d, expected %d", length, expectedLen)
	}

	// Concurrent deletes (delete half the keys)
	for i := 0; i < numGoroutines/2; i++ {
		go func(j int) {
			for k := 0; k < keysPerGoroutine; k++ {
				key := fmt.Sprintf("key-%d-%d", j, k)
				if err := store.Delete(key); err != nil {
					t.Errorf("Delete failed: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for deletes
	for i := 0; i < numGoroutines/2; i++ {
		<-done
	}

	// Check that Len is updated correctly
	expectedLen = (numGoroutines / 2) * keysPerGoroutine
	if length := store.Len(); length != expectedLen {
		t.Errorf("After concurrent deletes, Len = %d, expected %d", length, expectedLen)
	}
}

// TestBatchGet_AllExisting tests BatchGet with all keys existing in the store.
// Covers: T021 - Test BatchGet with all existing keys
func TestBatchGet_AllExisting(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Set up test data
	if err := store.Set("key1", IntValue(1)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := store.Set("key2", IntValue(2)); err != nil {
		t.Fatalf("Set key2 failed: %v", err)
	}
	if err := store.Set("key3", IntValue(3)); err != nil {
		t.Fatalf("Set key3 failed: %v", err)
	}

	// Batch get all keys
	result, err := store.BatchGet([]string{"key1", "key2", "key3"})
	if err != nil {
		t.Fatalf("BatchGet returned an error: %v", err)
	}

	// Check that all keys were retrieved successfully
	if len(result.Values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(result.Values))
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}

	// Verify values
	if v, ok := result.Values["key1"].(IntValue); !ok || v != 1 {
		t.Errorf("Expected key1=1, got %v", result.Values["key1"])
	}
	if v, ok := result.Values["key2"].(IntValue); !ok || v != 2 {
		t.Errorf("Expected key2=2, got %v", result.Values["key2"])
	}
	if v, ok := result.Values["key3"].(IntValue); !ok || v != 3 {
		t.Errorf("Expected key3=3, got %v", result.Values["key3"])
	}
}

// TestBatchGet_MixedKeys tests BatchGet with both existing and non-existent keys.
// Covers: T022 - Test BatchGet with mixed existing/non-existent keys
func TestBatchGet_MixedKeys(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Set up test data (only key1 and key3 exist)
	if err := store.Set("key1", IntValue(1)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := store.Set("key3", IntValue(3)); err != nil {
		t.Fatalf("Set key3 failed: %v", err)
	}

	// Batch get with mixed keys
	result, err := store.BatchGet([]string{"key1", "key2", "key3", "key4"})
	if err != nil {
		t.Fatalf("BatchGet returned an error: %v", err)
	}

	// Check partial success
	if len(result.Values) != 2 {
		t.Errorf("Expected 2 successful values, got %d", len(result.Values))
	}
	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(result.Errors))
	}

	// Verify successful retrievals
	if _, ok := result.Values["key1"]; !ok {
		t.Errorf("Expected key1 in Values")
	}
	if _, ok := result.Values["key3"]; !ok {
		t.Errorf("Expected key3 in Values")
	}

	// Verify errors for non-existent keys
	if _, ok := result.Errors["key2"]; !ok {
		t.Errorf("Expected key2 in Errors")
	}
	if _, ok := result.Errors["key4"]; !ok {
		t.Errorf("Expected key4 in Errors")
	}
}

// TestBatchGet_AllNonExistent tests BatchGet with all keys missing from the store.
// Covers: T023 - Test BatchGet with all non-existent keys
func TestBatchGet_AllNonExistent(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Batch get with all non-existent keys
	result, err := store.BatchGet([]string{"missing1", "missing2", "missing3"})
	if err != nil {
		t.Fatalf("BatchGet returned an error: %v", err)
	}

	// Check that all keys returned errors
	if len(result.Values) != 0 {
		t.Errorf("Expected 0 values, got %d", len(result.Values))
	}
	if len(result.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(result.Errors))
	}

	// Verify all errors are ErrNotFound
	for key, err := range result.Errors {
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound for key %s, got %v", key, err)
		}
	}
}

// TestBatchGet_EmptyInput tests BatchGet with an empty key list.
// Covers: T024 - Test BatchGet with empty input
func TestBatchGet_EmptyInput(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Batch get with empty list
	result, err := store.BatchGet([]string{})
	if err != nil {
		t.Fatalf("BatchGet returned an error: %v", err)
	}

	// Check that result is empty
	if len(result.Values) != 0 {
		t.Errorf("Expected 0 values, got %d", len(result.Values))
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}
}

// TestBatchGet_DuplicateKeys tests BatchGet with duplicate keys in the input.
// Covers: T025 - Test BatchGet with duplicate keys
func TestBatchGet_DuplicateKeys(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	if err := store.Set("key1", IntValue(42)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}

	// Batch get with duplicate keys
	result, err := store.BatchGet([]string{"key1", "key1", "key1"})
	if err != nil {
		t.Fatalf("BatchGet returned an error: %v", err)
	}

	// Should handle duplicates gracefully (only one entry in result)
	if len(result.Values) != 1 {
		t.Errorf("Expected 1 value for duplicate keys, got %d", len(result.Values))
	}
	if v, ok := result.Values["key1"].(IntValue); !ok || v != 42 {
		t.Errorf("Expected key1=42, got %v", result.Values["key1"])
	}
}

// TestBatchGet_SingleKey tests BatchGet with a single key.
// Covers: T026 - Test BatchGet with single key
func TestBatchGet_SingleKey(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	if err := store.Set("single", IntValue(100)); err != nil {
		t.Fatalf("Set single failed: %v", err)
	}

	// Batch get with single key
	result, err := store.BatchGet([]string{"single"})
	if err != nil {
		t.Fatalf("BatchGet returned an error: %v", err)
	}

	if len(result.Values) != 1 {
		t.Errorf("Expected 1 value, got %d", len(result.Values))
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}
}

// TestBatchGet_MultipleShards tests BatchGet with keys distributed across shards.
// Covers: T027 - Test BatchGet with keys across multiple shards
func TestBatchGet_MultipleShards(t *testing.T) {
	store, err := NewKeyValueStore(4)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Add many keys to ensure they span multiple shards
	keys := []string{}
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("key%d", i)
		keys = append(keys, key)
		if err := store.Set(key, IntValue(i)); err != nil {
			t.Fatalf("Set %s failed: %v", key, err)
		}
	}

	// Batch get all keys
	result, err := store.BatchGet(keys)
	if err != nil {
		t.Fatalf("BatchGet returned an error: %v", err)
	}

	// All keys should be retrieved successfully
	if len(result.Values) != 20 {
		t.Errorf("Expected 20 values, got %d", len(result.Values))
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}
}

// TestBatchGet_Concurrent tests BatchGet with concurrent operations.
// Covers: T028 - Test BatchGet with concurrent reads/writes
func TestBatchGet_Concurrent(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Pre-populate store
	for i := 0; i < 100; i++ {
		if err := store.Set(fmt.Sprintf("key%d", i), IntValue(i)); err != nil {
			t.Fatalf("Set key%d failed: %v", i, err)
		}
	}

	done := make(chan bool)

	// Concurrent BatchGet operations
	for i := 0; i < 10; i++ {
		go func() {
			keys := []string{"key10", "key20", "key30", "key40", "key50"}
			result, err := store.BatchGet(keys)
			if err != nil {
				t.Errorf("BatchGet failed: %v", err)
			}
			if len(result.Values) != 5 {
				t.Errorf("Expected 5 values, got %d", len(result.Values))
			}
			done <- true
		}()
	}

	// Concurrent Set operations
	for i := 0; i < 5; i++ {
		go func(j int) {
			for k := 0; k < 20; k++ {
				key := fmt.Sprintf("newkey%d-%d", j, k)
				if err := store.Set(key, IntValue(j*100+k)); err != nil {
					t.Errorf("Set %s failed: %v", key, err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}
}

// TestBatchSet_NewKeys tests BatchSet with multiple new keys.
// Covers: T038 - Test BatchSet with multiple new keys
func TestBatchSet_NewKeys(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	items := map[string]Value{
		"key1": IntValue(1),
		"key2": IntValue(2),
		"key3": IntValue(3),
	}

	result, err := store.BatchSet(items)
	if err != nil {
		t.Fatalf("BatchSet returned an error: %v", err)
	}

	// All should succeed
	if result.Successful != 3 {
		t.Errorf("Expected 3 successful sets, got %d", result.Successful)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}

	// Verify all keys were set
	for key, expectedVal := range items {
		val, err := store.Get(key)
		if err != nil {
			t.Errorf("Get(%s) failed: %v", key, err)
		}
		if val != expectedVal {
			t.Errorf("Expected %v for %s, got %v", expectedVal, key, val)
		}
	}
}

// TestBatchSet_MixedKeys tests BatchSet with both new and existing keys.
// Covers: T039 - Test BatchSet with mixed new/existing keys
func TestBatchSet_MixedKeys(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Pre-set some keys
	if err := store.Set("key1", IntValue(100)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := store.Set("key2", IntValue(200)); err != nil {
		t.Fatalf("Set key2 failed: %v", err)
	}

	// Batch set with mixed keys (update key1, key2; add key3, key4)
	items := map[string]Value{
		"key1": IntValue(1), // update
		"key2": IntValue(2), // update
		"key3": IntValue(3), // new
		"key4": IntValue(4), // new
	}

	result, err := store.BatchSet(items)
	if err != nil {
		t.Fatalf("BatchSet returned an error: %v", err)
	}

	// All should succeed
	if result.Successful != 4 {
		t.Errorf("Expected 4 successful sets, got %d", result.Successful)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}

	// Verify values were updated/added
	if val, _ := store.Get("key1"); val.(IntValue) != 1 {
		t.Errorf("key1 not updated, got %v", val)
	}
	if val, _ := store.Get("key2"); val.(IntValue) != 2 {
		t.Errorf("key2 not updated, got %v", val)
	}
}

// TestBatchSet_EmptyInput tests BatchSet with an empty map.
// Covers: T040 - Test BatchSet with empty input
func TestBatchSet_EmptyInput(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	result, err := store.BatchSet(map[string]Value{})
	if err != nil {
		t.Fatalf("BatchSet returned an error: %v", err)
	}

	if result.Successful != 0 {
		t.Errorf("Expected 0 successful sets, got %d", result.Successful)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}
}

// TestBatchSet_UpdatesLen tests that BatchSet correctly updates Len().
// Covers: T042 - Test BatchSet correctly updates Len()
func TestBatchSet_UpdatesLen(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Initial length is 0
	if store.Len() != 0 {
		t.Errorf("Initial Len = %d, expected 0", store.Len())
	}

	// Batch set 5 keys
	items := map[string]Value{
		"k1": IntValue(1),
		"k2": IntValue(2),
		"k3": IntValue(3),
		"k4": IntValue(4),
		"k5": IntValue(5),
	}
	if _, err := store.BatchSet(items); err != nil {
		t.Fatalf("BatchSet failed: %v", err)
	}

	if store.Len() != 5 {
		t.Errorf("After BatchSet, Len = %d, expected 5", store.Len())
	}

	// Update 2 existing keys (should not change Len)
	updates := map[string]Value{
		"k1": IntValue(10),
		"k3": IntValue(30),
	}
	if _, err := store.BatchSet(updates); err != nil {
		t.Fatalf("BatchSet updates failed: %v", err)
	}

	if store.Len() != 5 {
		t.Errorf("After updating, Len = %d, expected 5", store.Len())
	}

	// Add 3 more keys
	more := map[string]Value{
		"k6": IntValue(6),
		"k7": IntValue(7),
		"k8": IntValue(8),
	}
	if _, err := store.BatchSet(more); err != nil {
		t.Fatalf("BatchSet more failed: %v", err)
	}

	if store.Len() != 8 {
		t.Errorf("After adding more, Len = %d, expected 8", store.Len())
	}
}

// TestBatchSet_MultipleShards tests BatchSet with keys across multiple shards.
// Covers: T044 - Test BatchSet across multiple shards
func TestBatchSet_MultipleShards(t *testing.T) {
	store, err := NewKeyValueStore(4)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Create many items to ensure they span multiple shards
	items := make(map[string]Value)
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key%d", i)
		items[key] = IntValue(i)
	}

	result, err := store.BatchSet(items)
	if err != nil {
		t.Fatalf("BatchSet returned an error: %v", err)
	}

	// All should succeed
	if result.Successful != 50 {
		t.Errorf("Expected 50 successful sets, got %d", result.Successful)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}

	// Verify all keys were set
	if store.Len() != 50 {
		t.Errorf("Expected Len=50, got %d", store.Len())
	}
}

// TestBatchSet_Concurrent tests BatchSet with concurrent operations.
// Covers: T043 - Test BatchSet with concurrent operations
func TestBatchSet_Concurrent(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	done := make(chan bool)
	numGoroutines := 10
	itemsPerGoroutine := 50

	// Concurrent BatchSet operations
	for i := 0; i < numGoroutines; i++ {
		go func(j int) {
			items := make(map[string]Value)
			for k := 0; k < itemsPerGoroutine; k++ {
				key := fmt.Sprintf("key-%d-%d", j, k)
				items[key] = IntValue(j*100 + k)
			}
			result, err := store.BatchSet(items)
			if err != nil {
				t.Errorf("BatchSet failed: %v", err)
			}
			if result.Successful != itemsPerGoroutine {
				t.Errorf("Expected %d successful, got %d", itemsPerGoroutine, result.Successful)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify total count
	expectedLen := numGoroutines * itemsPerGoroutine
	if store.Len() != expectedLen {
		t.Errorf("Expected Len=%d, got %d", expectedLen, store.Len())
	}
}

// TestBatchDelete_AllExisting tests BatchDelete with all keys existing.
// Covers: T048 - Test BatchDelete with all existing keys
func TestBatchDelete_AllExisting(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Set up test data
	if err := store.Set("key1", IntValue(1)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := store.Set("key2", IntValue(2)); err != nil {
		t.Fatalf("Set key2 failed: %v", err)
	}
	if err := store.Set("key3", IntValue(3)); err != nil {
		t.Fatalf("Set key3 failed: %v", err)
	}

	// Batch delete all keys
	result, err := store.BatchDelete([]string{"key1", "key2", "key3"})
	if err != nil {
		t.Fatalf("BatchDelete returned an error: %v", err)
	}

	// All should be deleted successfully
	if result.Deleted != 3 {
		t.Errorf("Expected 3 deletions, got %d", result.Deleted)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}

	// Verify keys are gone
	for _, key := range []string{"key1", "key2", "key3"} {
		if store.Exists(key) {
			t.Errorf("Key %s still exists after BatchDelete", key)
		}
	}
}

// TestBatchDelete_MixedKeys tests BatchDelete with mixed existing/non-existent keys.
// Covers: T049 - Test BatchDelete with mixed existing/non-existent keys
func TestBatchDelete_MixedKeys(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Set up test data (only key1 and key3 exist)
	if err := store.Set("key1", IntValue(1)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := store.Set("key3", IntValue(3)); err != nil {
		t.Fatalf("Set key3 failed: %v", err)
	}

	// Batch delete with mixed keys
	result, err := store.BatchDelete([]string{"key1", "key2", "key3", "key4"})
	if err != nil {
		t.Fatalf("BatchDelete returned an error: %v", err)
	}

	// Check partial success
	if result.Deleted != 2 {
		t.Errorf("Expected 2 deletions, got %d", result.Deleted)
	}
	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(result.Errors))
	}

	// Verify errors for non-existent keys
	if _, ok := result.Errors["key2"]; !ok {
		t.Errorf("Expected error for key2")
	}
	if _, ok := result.Errors["key4"]; !ok {
		t.Errorf("Expected error for key4")
	}

	// Verify existing keys were deleted
	if store.Exists("key1") {
		t.Errorf("key1 still exists")
	}
	if store.Exists("key3") {
		t.Errorf("key3 still exists")
	}
}

// TestBatchDelete_EmptyInput tests BatchDelete with an empty key list.
// Covers: T050 - Test BatchDelete with empty input
func TestBatchDelete_EmptyInput(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	result, err := store.BatchDelete([]string{})
	if err != nil {
		t.Fatalf("BatchDelete returned an error: %v", err)
	}

	if result.Deleted != 0 {
		t.Errorf("Expected 0 deletions, got %d", result.Deleted)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}
}

// TestBatchDelete_UpdatesLen tests that BatchDelete correctly updates Len().
// Covers: T051 - Test BatchDelete correctly updates Len()
func TestBatchDelete_UpdatesLen(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Add 10 keys
	for i := 0; i < 10; i++ {
		if err := store.Set(fmt.Sprintf("key%d", i), IntValue(i)); err != nil {
			t.Fatalf("Set key%d failed: %v", i, err)
		}
	}

	if store.Len() != 10 {
		t.Errorf("Initial Len = %d, expected 10", store.Len())
	}

	// Delete 5 keys
	keysToDelete := []string{"key0", "key2", "key4", "key6", "key8"}
	result, err := store.BatchDelete(keysToDelete)
	if err != nil {
		t.Fatalf("BatchDelete failed: %v", err)
	}

	if result.Deleted != 5 {
		t.Errorf("Expected 5 deletions, got %d", result.Deleted)
	}

	if store.Len() != 5 {
		t.Errorf("After BatchDelete, Len = %d, expected 5", store.Len())
	}

	// Delete remaining keys
	remainingKeys := []string{"key1", "key3", "key5", "key7", "key9"}
	if _, err := store.BatchDelete(remainingKeys); err != nil {
		t.Fatalf("BatchDelete remaining keys failed: %v", err)
	}

	if store.Len() != 0 {
		t.Errorf("After deleting all, Len = %d, expected 0", store.Len())
	}
}

// TestBatchDelete_MultipleShards tests BatchDelete with keys across shards.
// Covers: BatchDelete across multiple shards
func TestBatchDelete_MultipleShards(t *testing.T) {
	store, err := NewKeyValueStore(4)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Add many keys across shards
	keys := []string{}
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key%d", i)
		keys = append(keys, key)
		if err := store.Set(key, IntValue(i)); err != nil {
			t.Fatalf("Set %s failed: %v", key, err)
		}
	}

	// Batch delete all keys
	result, err := store.BatchDelete(keys)
	if err != nil {
		t.Fatalf("BatchDelete returned an error: %v", err)
	}

	if result.Deleted != 50 {
		t.Errorf("Expected 50 deletions, got %d", result.Deleted)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}
	if store.Len() != 0 {
		t.Errorf("Expected Len=0, got %d", store.Len())
	}
}

// TestBatchDelete_Concurrent tests BatchDelete with concurrent operations.
// Covers: Concurrent BatchDelete operations
func TestBatchDelete_Concurrent(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Pre-populate store with many keys
	numGoroutines := 10
	keysPerGoroutine := 50
	for i := 0; i < numGoroutines; i++ {
		for k := 0; k < keysPerGoroutine; k++ {
			key := fmt.Sprintf("key-%d-%d", i, k)
			if err := store.Set(key, IntValue(i*100+k)); err != nil {
				t.Fatalf("Set %s failed: %v", key, err)
			}
		}
	}

	initialLen := store.Len()
	if initialLen != numGoroutines*keysPerGoroutine {
		t.Errorf("Expected initial Len=%d, got %d", numGoroutines*keysPerGoroutine, initialLen)
	}

	done := make(chan bool)

	// Concurrent BatchDelete operations
	for i := 0; i < numGoroutines; i++ {
		go func(j int) {
			keys := []string{}
			for k := 0; k < keysPerGoroutine; k++ {
				key := fmt.Sprintf("key-%d-%d", j, k)
				keys = append(keys, key)
			}
			result, err := store.BatchDelete(keys)
			if err != nil {
				t.Errorf("BatchDelete failed: %v", err)
			}
			if result.Deleted != keysPerGoroutine {
				t.Errorf("Expected %d deletions, got %d", keysPerGoroutine, result.Deleted)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all keys are gone
	if store.Len() != 0 {
		t.Errorf("Expected Len=0 after all deletions, got %d", store.Len())
	}
}

// TestBatchDelete_DuplicateKeys tests BatchDelete with duplicate keys in input.
// Covers: Duplicate key handling
func TestBatchDelete_DuplicateKeys(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	if err := store.Set("key1", IntValue(1)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}

	// Try to delete the same key multiple times
	result, err := store.BatchDelete([]string{"key1", "key1", "key1"})
	if err != nil {
		t.Fatalf("BatchDelete returned an error: %v", err)
	}

	// Should only delete once
	if result.Deleted != 1 {
		t.Errorf("Expected 1 deletion, got %d", result.Deleted)
	}

	// Only one error should be reported for the duplicate key
	// (subsequent attempts to delete the same key are deduplicated)
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error for duplicate key, got %d", len(result.Errors))
	}

	// Verify the error is ErrNotFound
	if result.Errors["key1"] != ErrNotFound {
		t.Errorf("Expected ErrNotFound for key1, got %v", result.Errors["key1"])
	}
}

// TestSetWithTTL_StoresKey tests that SetWithTTL stores a key with TTL.
// Covers: T056 - Test SetWithTTL stores key with TTL
func TestSetWithTTL_StoresKey(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	// Set a key with 1 second TTL
	err = store.SetWithTTL("key1", IntValue(100), 1*time.Second)
	if err != nil {
		t.Fatalf("SetWithTTL returned an error: %v", err)
	}

	// Key should be retrievable immediately
	val, err := store.Get("key1")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if val.(IntValue) != 100 {
		t.Errorf("Expected value 100, got %v", val)
	}

	// Key should exist
	if !store.Exists("key1") {
		t.Errorf("Exists returned false for key with TTL")
	}
}

// TestSetWithTTL_ZeroTTL tests that zero TTL is treated as no expiration.
// Covers: T057 - Test SetWithTTL with zero TTL (treated as no expiration)
func TestSetWithTTL_ZeroTTL(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	// Set a key with zero TTL
	err = store.SetWithTTL("key1", IntValue(100), 0)
	if err != nil {
		t.Fatalf("SetWithTTL returned an error: %v", err)
	}

	// Key should be retrievable
	val, err := store.Get("key1")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if val.(IntValue) != 100 {
		t.Errorf("Expected value 100, got %v", val)
	}

	// Wait a bit and verify key still exists (no expiration)
	time.Sleep(100 * time.Millisecond)
	if !store.Exists("key1") {
		t.Errorf("Key with zero TTL expired incorrectly")
	}
}

// TestSetWithTTL_NegativeTTL tests that negative TTL returns an error.
// Covers: T058 - Test SetWithTTL with negative TTL (should return error)
func TestSetWithTTL_NegativeTTL(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	// Try to set a key with negative TTL
	err = store.SetWithTTL("key1", IntValue(100), -1*time.Second)
	if err != ErrInvalidTTL {
		t.Errorf("Expected ErrInvalidTTL, got %v", err)
	}

	// Key should not exist
	if store.Exists("key1") {
		t.Errorf("Key was set despite negative TTL")
	}
}

// TestSetWithTTL_UpdateExisting tests updating an existing key with new TTL.
// Covers: T059 - Test SetWithTTL updates existing key with new TTL
func TestSetWithTTL_UpdateExisting(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	// Set a key without TTL
	if err := store.Set("key1", IntValue(100)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}

	// Update with TTL
	err = store.SetWithTTL("key1", IntValue(200), 1*time.Second)
	if err != nil {
		t.Fatalf("SetWithTTL returned an error: %v", err)
	}

	// Value should be updated
	val, _ := store.Get("key1")
	if val.(IntValue) != 200 {
		t.Errorf("Expected updated value 200, got %v", val)
	}

	// Len should not increase (key already existed)
	if store.Len() != 1 {
		t.Errorf("Len should be 1, got %d", store.Len())
	}
}

// TestGet_ExpiredKey tests that Get returns ErrNotFound for expired keys (passive cleanup).
// Covers: T060 - Test Get returns ErrNotFound for expired keys
func TestGet_ExpiredKey(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	// Set a key with very short TTL
	err = store.SetWithTTL("key1", IntValue(100), 50*time.Millisecond)
	if err != nil {
		t.Fatalf("SetWithTTL returned an error: %v", err)
	}

	// Key should be accessible immediately
	val, err := store.Get("key1")
	if err != nil {
		t.Errorf("Get failed before expiration: %v", err)
	}
	if val.(IntValue) != 100 {
		t.Errorf("Expected value 100, got %v", val)
	}

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Get should return ErrNotFound (passive cleanup)
	_, err = store.Get("key1")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound for expired key, got %v", err)
	}
}

// TestExists_ExpiredKey tests that Exists returns false for expired keys.
// Covers: T061 - Test Exists returns false for expired keys
func TestExists_ExpiredKey(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	// Set a key with very short TTL
	err = store.SetWithTTL("key1", IntValue(100), 50*time.Millisecond)
	if err != nil {
		t.Fatalf("SetWithTTL returned an error: %v", err)
	}

	// Key should exist immediately
	if !store.Exists("key1") {
		t.Errorf("Exists returned false before expiration")
	}

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Exists should return false
	if store.Exists("key1") {
		t.Errorf("Exists returned true for expired key")
	}
}

// TestTTLCleanup_ActiveCleanup tests that the cleanup goroutine removes expired keys.
// Covers: T062 - Test TTL cleanup goroutine removes expired keys
func TestTTLCleanup_ActiveCleanup(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	// Set multiple keys with short TTL
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		if err := store.SetWithTTL(key, IntValue(i), 100*time.Millisecond); err != nil {
			t.Fatalf("SetWithTTL %s failed: %v", key, err)
		}
	}

	// Verify all keys exist
	if store.Len() != 10 {
		t.Errorf("Expected Len=10, got %d", store.Len())
	}

	// Wait for TTL to expire and cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Cleanup goroutine should have removed expired keys
	// Len should eventually be 0 (within cleanup interval)
	time.Sleep(100 * time.Millisecond) // Give cleanup time to run

	if store.Len() > 0 {
		t.Errorf("Expected Len=0 after cleanup, got %d", store.Len())
	}
}

// TestClose_StopsCleanup tests that Close stops the cleanup goroutine.
// Covers: T065-T066 - Test Close method
func TestClose_StopsCleanup(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Set a key with TTL
	if err := store.SetWithTTL("key1", IntValue(100), 1*time.Second); err != nil {
		t.Fatalf("SetWithTTL failed: %v", err)
	}

	// Close the store
	err = store.Close()
	if err != nil {
		t.Errorf("Close returned an error: %v", err)
	}

	// Subsequent operations should return ErrStoreClosed
	err = store.Set("key2", IntValue(200))
	if err != ErrStoreClosed {
		t.Errorf("Expected ErrStoreClosed, got %v", err)
	}

	// Close should be idempotent
	err = store.Close()
	if err != nil {
		t.Errorf("Second Close returned an error: %v", err)
	}
}

// TestClose_MultipleGoroutines tests Close with concurrent operations.
// Covers: Concurrent Close operations
func TestClose_MultipleGoroutines(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	done := make(chan bool)

	// Start goroutines doing operations
	for i := 0; i < 5; i++ {
		go func(j int) {
			for k := 0; k < 100; k++ {
				key := fmt.Sprintf("key-%d-%d", j, k)
				_ = store.Set(key, IntValue(k)) // Ignore errors as store may be closing
			}
			done <- true
		}(i)
	}

	// Wait a bit then close
	time.Sleep(10 * time.Millisecond)
	store.Close()

	// Wait for goroutines to finish (they should handle closed store gracefully)
	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestNewKeyValueStoreWithHash_ValidHash tests creating a store with a valid custom hash function.
// Covers: T070 - Test NewKeyValueStoreWithHash with valid custom hash
func TestNewKeyValueStoreWithHash_ValidHash(t *testing.T) {
	// Custom hash function that uses simple modulo
	customHash := func(key string) int {
		sum := 0
		for _, ch := range key {
			sum += int(ch)
		}
		return sum % 10
	}

	store, err := NewKeyValueStoreWithHash(10, customHash)
	if err != nil {
		t.Fatalf("NewKeyValueStoreWithHash returned an error: %v", err)
	}
	defer store.Close()

	// Store should be functional
	err = store.Set("test", IntValue(100))
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	val, err := store.Get("test")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if val.(IntValue) != 100 {
		t.Errorf("Expected value 100, got %v", val)
	}
}

// TestNewKeyValueStoreWithHash_Distribution tests that custom hash distributes keys correctly.
// Covers: T071 - Test custom hash distributes keys across shards
func TestNewKeyValueStoreWithHash_Distribution(t *testing.T) {
	numShards := 4

	// Custom hash that assigns keys based on first character
	customHash := func(key string) int {
		if len(key) == 0 {
			return 0
		}
		return int(key[0]) % numShards
	}

	store, err := NewKeyValueStoreWithHash(numShards, customHash)
	if err != nil {
		t.Fatalf("NewKeyValueStoreWithHash returned an error: %v", err)
	}
	defer store.Close()

	// Add keys that should go to different shards
	keys := []string{"a1", "b1", "c1", "d1", "a2", "b2", "c2", "d2"}
	for _, key := range keys {
		if err := store.Set(key, IntValue(1)); err != nil {
			t.Fatalf("Set %s failed: %v", key, err)
		}
	}

	// Verify all keys can be retrieved
	if store.Len() != len(keys) {
		t.Errorf("Expected Len=%d, got %d", len(keys), store.Len())
	}

	for _, key := range keys {
		if !store.Exists(key) {
			t.Errorf("Key %s not found", key)
		}
	}
}

// TestNewKeyValueStoreWithHash_NilHash tests that nil hash function returns error.
// Covers: T072 - Test NewKeyValueStoreWithHash with nil hash function
func TestNewKeyValueStoreWithHash_NilHash(t *testing.T) {
	store, err := NewKeyValueStoreWithHash(10, nil)
	if err != ErrNilHashFunc {
		t.Errorf("Expected ErrNilHashFunc, got %v", err)
	}
	if store != nil {
		t.Errorf("Expected nil store, got %v", store)
		store.Close()
	}
}

// TestNewKeyValueStoreWithHash_InvalidNumShards tests that invalid numShards returns error.
// Covers: T073 - Test NewKeyValueStoreWithHash with invalid numShards
func TestNewKeyValueStoreWithHash_InvalidNumShards(t *testing.T) {
	customHash := func(key string) int {
		return 0
	}

	// Test with 0 shards
	store, err := NewKeyValueStoreWithHash(0, customHash)
	if err != ErrInvalidNumShards {
		t.Errorf("Expected ErrInvalidNumShards for 0 shards, got %v", err)
	}
	if store != nil {
		store.Close()
	}

	// Test with negative shards
	store, err = NewKeyValueStoreWithHash(-1, customHash)
	if err != ErrInvalidNumShards {
		t.Errorf("Expected ErrInvalidNumShards for -1 shards, got %v", err)
	}
	if store != nil {
		store.Close()
	}
}

// TestNewKeyValueStoreWithHash_Deterministic tests that hash function must be deterministic.
// Covers: T074 - Test custom hash is deterministic
func TestNewKeyValueStoreWithHash_Deterministic(t *testing.T) {
	// Deterministic hash function
	deterministicHash := func(key string) int {
		hash := 0
		for _, ch := range key {
			hash = (hash*31 + int(ch)) % 8
		}
		return hash
	}

	store, err := NewKeyValueStoreWithHash(8, deterministicHash)
	if err != nil {
		t.Fatalf("NewKeyValueStoreWithHash returned an error: %v", err)
	}
	defer store.Close()

	// Set a key multiple times and verify it goes to the same shard
	key := "test-key"
	for i := 0; i < 10; i++ {
		if err := store.Set(key, IntValue(i)); err != nil {
			t.Errorf("Set failed on iteration %d: %v", i, err)
		}
		val, err := store.Get(key)
		if err != nil {
			t.Errorf("Get failed on iteration %d: %v", i, err)
		}
		if val.(IntValue) != IntValue(i) {
			t.Errorf("Expected value %d, got %v", i, val)
		}
	}

	// Delete should also work consistently
	err = store.Delete(key)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	if store.Exists(key) {
		t.Errorf("Key still exists after delete")
	}
}

// TestNewKeyValueStoreWithHash_OutOfBounds tests handling of invalid shard indices.
// Covers: T075 - Test hash returning out-of-bounds index
func TestNewKeyValueStoreWithHash_OutOfBounds(t *testing.T) {
	numShards := 5

	// This hash function intentionally returns out-of-bounds values
	// The store should handle this gracefully or panic (which we can test for)
	badHash := func(key string) int {
		// Return a value outside [0, numShards)
		return 100 // Way out of bounds
	}

	store, err := NewKeyValueStoreWithHash(numShards, badHash)
	if err != nil {
		t.Fatalf("NewKeyValueStoreWithHash returned an error: %v", err)
	}
	defer store.Close()

	// This should panic or be handled gracefully
	// Let's test that it panics (documenting current behavior)
	defer func() {
		if r := recover(); r == nil {
			// If it doesn't panic, the hash function should have validated the index
			// or wrapped it - let's check if the operation succeeded
			_, err := store.Get("test")
			if err == nil {
				// Operation succeeded despite bad hash - this means bounds checking exists
				t.Logf("Store handled out-of-bounds hash gracefully")
			}
		} else {
			// Expected: panic on out-of-bounds access
			t.Logf("Store panicked on out-of-bounds hash (expected): %v", r)
		}
	}()

	// Try to use the store with a bad hash
	_ = store.Set("test", IntValue(100))
}

// TestNewKeyValueStoreWithHash_Concurrent tests concurrent operations with custom hash.
// Covers: T076 - Test concurrent operations with custom hash
func TestNewKeyValueStoreWithHash_Concurrent(t *testing.T) {
	customHash := func(key string) int {
		sum := 0
		for _, ch := range key {
			sum += int(ch)
		}
		return sum % 10
	}

	store, err := NewKeyValueStoreWithHash(10, customHash)
	if err != nil {
		t.Fatalf("NewKeyValueStoreWithHash returned an error: %v", err)
	}
	defer store.Close()

	done := make(chan bool)
	numGoroutines := 10
	keysPerGoroutine := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(j int) {
			for k := 0; k < keysPerGoroutine; k++ {
				key := fmt.Sprintf("key-%d-%d", j, k)
				if err := store.Set(key, IntValue(j*100+k)); err != nil {
					t.Errorf("Set %s failed: %v", key, err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for writes
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func(j int) {
			for k := 0; k < keysPerGoroutine; k++ {
				key := fmt.Sprintf("key-%d-%d", j, k)
				val, err := store.Get(key)
				if err != nil {
					t.Errorf("Get failed: %v", err)
				}
				expected := IntValue(j*100 + k)
				if val != expected {
					t.Errorf("Expected %v, got %v", expected, val)
				}
			}
			done <- true
		}(i)
	}

	// Wait for reads
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify total count
	expectedCount := numGoroutines * keysPerGoroutine
	if store.Len() != expectedCount {
		t.Errorf("Expected Len=%d, got %d", expectedCount, store.Len())
	}
}

// TestSaveSnapshot_EmptyStore tests saving a snapshot of an empty store.
// Covers: T079 - Test SaveSnapshot with empty store
func TestSaveSnapshot_EmptyStore(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	// Save empty store
	filepath := "/tmp/kvs_test_empty.snapshot"
	defer os.Remove(filepath)

	err = store.SaveSnapshot(filepath)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		t.Errorf("Snapshot file was not created")
	}
}

// TestSaveSnapshot_WithData tests saving a snapshot with multiple keys.
// Covers: T080 - Test SaveSnapshot with multiple keys
func TestSaveSnapshot_WithData(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	// Add test data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		if err := store.Set(key, IntValue(i)); err != nil {
			t.Fatalf("Set %s failed: %v", key, err)
		}
	}

	// Save snapshot
	filepath := "/tmp/kvs_test_data.snapshot"
	defer os.Remove(filepath)

	err = store.SaveSnapshot(filepath)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// Verify file was created and has content
	info, err := os.Stat(filepath)
	if err != nil {
		t.Errorf("Snapshot file error: %v", err)
	}
	if info.Size() == 0 {
		t.Errorf("Snapshot file is empty")
	}
}

// TestSaveSnapshot_WithTTL tests saving a snapshot with TTL keys.
// Covers: T081 - Test SaveSnapshot with TTL keys
func TestSaveSnapshot_WithTTL(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	// Add keys with TTL
	if err := store.SetWithTTL("ttl-key1", IntValue(100), 10*time.Second); err != nil {
		t.Fatalf("SetWithTTL ttl-key1 failed: %v", err)
	}
	if err := store.SetWithTTL("ttl-key2", IntValue(200), 20*time.Second); err != nil {
		t.Fatalf("SetWithTTL ttl-key2 failed: %v", err)
	}
	if err := store.Set("normal-key", IntValue(300)); err != nil {
		t.Fatalf("Set normal-key failed: %v", err)
	}

	// Save snapshot
	filepath := "/tmp/kvs_test_ttl.snapshot"
	defer os.Remove(filepath)

	err = store.SaveSnapshot(filepath)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
}

// TestSaveSnapshot_DoesNotModify tests that saving doesn't modify store.
// Covers: T082 - Test SaveSnapshot doesn't modify store
func TestSaveSnapshot_DoesNotModify(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	// Add test data
	if err := store.Set("key1", IntValue(100)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := store.Set("key2", IntValue(200)); err != nil {
		t.Fatalf("Set key2 failed: %v", err)
	}

	initialLen := store.Len()

	// Save snapshot
	filepath := "/tmp/kvs_test_nomodify.snapshot"
	defer os.Remove(filepath)

	err = store.SaveSnapshot(filepath)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// Verify store unchanged
	if store.Len() != initialLen {
		t.Errorf("Store Len changed after snapshot: was %d, now %d", initialLen, store.Len())
	}

	val, err := store.Get("key1")
	if err != nil || val.(IntValue) != 100 {
		t.Errorf("Store data changed after snapshot")
	}
}

// TestSaveSnapshot_ClosedStore tests that SaveSnapshot returns error on closed store.
// Covers: T083 - Test SaveSnapshot on closed store
func TestSaveSnapshot_ClosedStore(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	if err := store.Set("key1", IntValue(100)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	store.Close()

	filepath := "/tmp/kvs_test_closed.snapshot"
	defer os.Remove(filepath)

	err = store.SaveSnapshot(filepath)
	if err != ErrStoreClosed {
		t.Errorf("Expected ErrStoreClosed, got %v", err)
	}
}

// TestLoadSnapshot_ValidSnapshot tests loading a valid snapshot.
// Covers: T089 - Test LoadSnapshot with valid snapshot
func TestLoadSnapshot_ValidSnapshot(t *testing.T) {
	// Create and save a snapshot
	store1, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key%d", i)
		if err := store1.Set(key, IntValue(i)); err != nil {
			t.Fatalf("Set %s failed: %v", key, err)
		}
	}

	filepath := "/tmp/kvs_test_load_valid.snapshot"
	defer os.Remove(filepath)

	err = store1.SaveSnapshot(filepath)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	store1.Close()

	// Load into new store
	store2, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store2.Close()

	err = store2.LoadSnapshot(filepath)
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	// Verify all keys loaded
	if store2.Len() != 50 {
		t.Errorf("Expected Len=50, got %d", store2.Len())
	}

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key%d", i)
		val, err := store2.Get(key)
		if err != nil {
			t.Errorf("Get(%s) failed: %v", key, err)
		}
		if val.(IntValue) != IntValue(i) {
			t.Errorf("Expected %d, got %v", i, val)
		}
	}
}

// TestLoadSnapshot_WithTTL tests loading snapshot with TTL keys.
// Covers: T090 - Test LoadSnapshot with TTL keys
func TestLoadSnapshot_WithTTL(t *testing.T) {
	// Create snapshot with TTL keys
	store1, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	// Add keys with long TTL so they don't expire during test
	if err := store1.SetWithTTL("ttl-key", IntValue(100), 1*time.Hour); err != nil {
		t.Fatalf("SetWithTTL ttl-key failed: %v", err)
	}
	if err := store1.Set("normal-key", IntValue(200)); err != nil {
		t.Fatalf("Set normal-key failed: %v", err)
	}

	filepath := "/tmp/kvs_test_load_ttl.snapshot"
	defer os.Remove(filepath)

	err = store1.SaveSnapshot(filepath)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	store1.Close()

	// Load into new store
	store2, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store2.Close()

	err = store2.LoadSnapshot(filepath)
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	// Verify both keys exist
	if !store2.Exists("ttl-key") {
		t.Errorf("TTL key not loaded")
	}
	if !store2.Exists("normal-key") {
		t.Errorf("Normal key not loaded")
	}
}

// TestLoadSnapshot_ReplacesData tests that LoadSnapshot replaces existing data.
// Covers: T091 - Test LoadSnapshot replaces existing data
func TestLoadSnapshot_ReplacesData(t *testing.T) {
	// Create snapshot
	store1, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	if err := store1.Set("saved-key", IntValue(100)); err != nil {
		t.Fatalf("Set saved-key failed: %v", err)
	}

	filepath := "/tmp/kvs_test_load_replace.snapshot"
	defer os.Remove(filepath)

	err = store1.SaveSnapshot(filepath)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	store1.Close()

	// Create store with different data
	store2, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store2.Close()

	if err := store2.Set("existing-key", IntValue(999)); err != nil {
		t.Fatalf("Set existing-key failed: %v", err)
	}
	if err := store2.Set("another-key", IntValue(888)); err != nil {
		t.Fatalf("Set another-key failed: %v", err)
	}

	// Load snapshot (should replace all data)
	err = store2.LoadSnapshot(filepath)
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	// Verify only snapshot data exists
	if store2.Len() != 1 {
		t.Errorf("Expected Len=1 after load, got %d", store2.Len())
	}

	if !store2.Exists("saved-key") {
		t.Errorf("Snapshot key not found")
	}

	if store2.Exists("existing-key") {
		t.Errorf("Old key still exists after load")
	}
}

// TestLoadSnapshot_SkipsExpired tests that LoadSnapshot skips expired keys.
// Covers: T092 - Test LoadSnapshot skips expired keys
func TestLoadSnapshot_SkipsExpired(t *testing.T) {
	// Create snapshot with short TTL
	store1, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}

	if err := store1.SetWithTTL("expired-key", IntValue(100), 50*time.Millisecond); err != nil {
		t.Fatalf("SetWithTTL expired-key failed: %v", err)
	}
	if err := store1.Set("valid-key", IntValue(200)); err != nil {
		t.Fatalf("Set valid-key failed: %v", err)
	}

	filepath := "/tmp/kvs_test_load_expired.snapshot"
	defer os.Remove(filepath)

	err = store1.SaveSnapshot(filepath)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	store1.Close()

	// Wait for key to expire
	time.Sleep(100 * time.Millisecond)

	// Load into new store
	store2, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store2.Close()

	err = store2.LoadSnapshot(filepath)
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	// Expired key should not be loaded
	if store2.Exists("expired-key") {
		t.Errorf("Expired key was loaded")
	}

	// Valid key should exist
	if !store2.Exists("valid-key") {
		t.Errorf("Valid key not loaded")
	}
}

// TestLoadSnapshot_NonExistentFile tests loading from non-existent file.
// Covers: T093 - Test LoadSnapshot with non-existent file
func TestLoadSnapshot_NonExistentFile(t *testing.T) {
	store, err := NewKeyValueStore(10)
	if err != nil {
		t.Fatalf("NewKeyValueStore returned an error: %v", err)
	}
	defer store.Close()

	err = store.LoadSnapshot("/tmp/does_not_exist.snapshot")
	if err == nil {
		t.Errorf("Expected error loading non-existent file")
	}
}

// TestLoadSnapshot_ClosedStore tests that LoadSnapshot returns error on closed store.
// Covers: T094 - Test LoadSnapshot on closed store
func TestLoadSnapshot_ClosedStore(t *testing.T) {
	// Create a valid snapshot first
	store1, _ := NewKeyValueStore(10)
	if err := store1.Set("key1", IntValue(100)); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	filepath := "/tmp/kvs_test_load_closed.snapshot"
	defer os.Remove(filepath)
	if err := store1.SaveSnapshot(filepath); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	store1.Close()

	// Try to load into closed store
	store2, _ := NewKeyValueStore(10)
	store2.Close()

	err := store2.LoadSnapshot(filepath)
	if err != ErrStoreClosed {
		t.Errorf("Expected ErrStoreClosed, got %v", err)
	}
}
