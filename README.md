# kvs

kvs is a simple key-value store library for Go.

It offers the following functionality:

* Get: retrieve a value associated with a given key from the store
* Set: add or update a key-value pair in the store
* Delete: remove a key-value pair associated with a given key from the store
* Keys: retrieve a slice of all the keys in the store

This library defines two interfaces:

`Value` which defines the methods a value in the key-value store must implement

`Store` which defines the methods that a key-value store must implement

`ErrCode` defines an enumeration that represents the error codes that can be returned by the store.

The error codes are:

* `ErrUnknown`: represents an unknown error
* `ErrNotFound`: represents an error that occurs when the key is not found in the store
* `ErrDuplicate`: represents an error that occurs when the key already exists in the store

## Installation

Use `go get` to install kvs.

```bash
go get github.com/bay0/kvs
```

## Usage

To use the library, import the kvs package and create a new instance of the KeyValueStore using NewKeyValueStore().

```go
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

```
