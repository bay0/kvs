package main

import (
	"fmt"

	"github.com/bay0/kvs"
)

type Node struct {
	ID    int
	Store *kvs.KeyValueStore
}

type Cluster struct {
	Nodes []Node
}

func (c *Cluster) GetNode(key string) *Node {
	h := hash(key)
	idx := int(h % uint32(len(c.Nodes)))
	return &c.Nodes[idx]
}

type StringValue string

func (sv StringValue) Clone() kvs.Value {
	return sv
}

func main() {
	// Create a cluster of nodes
	store1, _ := kvs.NewKeyValueStore(16)

	store2, _ := kvs.NewKeyValueStore(16)

	store3, _ := kvs.NewKeyValueStore(16)

	cluster := &Cluster{
		Nodes: []Node{
			{ID: 1, Store: store1},
			{ID: 2, Store: store2},
			{ID: 3, Store: store3},
		},
	}

	// Add some key-value pairs to the store
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		node := cluster.GetNode(key)
		node.Store.Set(key, StringValue(value))
	}

	// Retrieve a value from the store
	key := "key-42"
	node := cluster.GetNode(key)
	val, err := node.Store.Get(key)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Value for %s: %s\n", key, val)

	// Delete a value from the store
	key = "key-73"
	node = cluster.GetNode(key)
	err = node.Store.Delete(key)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Deleted key: %s\n", key)

	// Get all keys from the store
	var keys []string
	for _, node := range cluster.Nodes {
		nodeKeys, _ := node.Store.Keys()
		keys = append(keys, nodeKeys...)
	}
	fmt.Println("Keys in the store:", keys)
}

func hash(key string) uint32 {
	// implement your own hashing algorithm here
	var h uint32
	for i := 0; i < len(key); i++ {
		h = 61*h + uint32(key[i])
	}

	return h
}
