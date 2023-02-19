package main

import (
	"fmt"
	"sync"

	"github.com/bay0/kvs"
)

type StringValue string

func (sv StringValue) Clone() kvs.Value {
	return sv
}

func main() {
	kv, err := kvs.NewKeyValueStore(512)
	if err != nil {
		fmt.Printf("Error creating KeyValueStore: %s\n", err.Error())
		return
	}

	var wg sync.WaitGroup

	// Spawn 10 worker threads to concurrently read from and write to the key-value store.
	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for j := 0; j < 1000000; j++ {
				// Generate a random key and value.
				key := fmt.Sprintf("key-%d-%d", id, j)
				val := fmt.Sprintf("val-%d-%d", id, j)

				// Write the key-value pair to the store.
				err := kv.Set(key, StringValue(val))
				if err != nil {
					fmt.Printf("Error setting key-value pair: %s\n", err.Error())
				}

				// Read the value back from the store.
				v, err := kv.Get(key)
				if err != nil {
					fmt.Printf("Error getting value for key %s: %s\n", key, err.Error())
				} else {
					sv, ok := v.(StringValue)
					if !ok {
						fmt.Printf("Value for key %s is not a string value\n", key)
					} else if sv.Clone().(StringValue) != StringValue(val) {
						fmt.Printf("Value for key %s does not match expected value\n", key)
					}
				}
			}
		}(i)
	}

	// Wait for all worker threads to complete.
	wg.Wait()

	// Print the size of the key-value store.
	fmt.Printf("Size of key-value store: %s\n", kv.Size())
}
