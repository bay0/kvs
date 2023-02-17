package main

import (
	"fmt"

	"github.com/bay0/kvs"
)

type Person struct {
	Name string
	Age  int
}

func (p *Person) Clone() kvs.Value {
	return &Person{
		Name: p.Name,
		Age:  p.Age,
	}
}

func main() {
	// Create a new key-value store
	store := kvs.NewKeyValueStore()

	// Create a new person value
	person := &Person{
		Name: "John",
		Age:  20,
	}

	// Set the person value in the store
	err := store.Set("person", person)
	if err != nil {
		// Handle the error
	}

	// Get the person value from the store
	val, err := store.Get("person")
	if err != nil {
		// Handle the error
	}

	// Cast the value to a person object
	personVal, ok := val.(*Person)
	if !ok {
		// Handle the error
	}

	// Print the person's name and age
	fmt.Printf("%s is %d years old.\n", personVal.Name, personVal.Age)

	// Delete the person value from the store
	err = store.Delete("person")
	if err != nil {
		// Handle the error
	}

	// Get all keys from the store
	keys := store.Keys()
	fmt.Println("Keys in the store:", keys)
}
