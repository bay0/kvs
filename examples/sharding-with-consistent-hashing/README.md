# Sharding with Consistent Hashing

This example shows how to shard a key-value store across multiple nodes using consistent hashing.

```go
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
 cluster := &Cluster{
  Nodes: []Node{
   {ID: 1, Store: kvs.NewKeyValueStore(2)},
   {ID: 2, Store: kvs.NewKeyValueStore(2)},
   {ID: 3, Store: kvs.NewKeyValueStore(2)},
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
 // implement your own hashing algorithm
 return 0
}
```

In this example, we define a Node struct that contains an ID and a Store object, which is an instance of the kvs.Store interface.

We also define a Cluster struct that contains a slice of Node objects. The GetNode method of the Cluster object uses consistent hashing to determine which node a given key should be stored on.

We then create a cluster of three nodes and add 100 key-value pairs to the store using the Set method.

To retrieve a value from the store, we call the Get method of the node that the key is mapped to using the GetNode method.

To delete a value from the store, we call the Delete method of the node that the key is mapped to using the GetNode method.

To get all keys from the store, we iterate over all nodes in the cluster and call the Keys method of each node's store.
