package main

import (
	"context"
	"log"
	"math/rand"
	"strconv"
	"time"

	dapr "github.com/dapr/go-sdk/client"
)

const componentName = "mysqlite"

func main() {
	client, err := dapr.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Dapr client: %v", err)
	}
	defer client.Close()

	rand.Seed(time.Now().UnixMicro())
	for i := 0; i < 10; i++ {
		orderId := rand.Intn(1000-1) + 1

		ctx := context.Background()

		err = client.SaveState(ctx, componentName, "order_1", []byte(strconv.Itoa(orderId)))
		if err != nil {
			panic(err)
		}

		result, err := client.GetState(ctx, componentName, "order_1")
		if err != nil {
			panic(err)
		}

		log.Println("Result after get:", string(result.Value))
		time.Sleep(2 * time.Second)
	}
}
