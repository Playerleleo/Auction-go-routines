package auction

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestAutoCloseExpiredAuction(t *testing.T) {
	// Setup
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://admin:admin@localhost:27017/auctions_test?authSource=admin"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(ctx)

	repo := NewAuctionRepository(client.Database("auctions_test"))
	repo.Collection.Drop(ctx)

	// Criar um leilão diretamente no MongoDB
	auctionDoc := bson.D{
		{Key: "_id", Value: "test-auction-1"},
		{Key: "product_name", Value: "Test Product"},
		{Key: "category", Value: "Test Category"},
		{Key: "description", Value: "Test Description"},
		{Key: "condition", Value: auction_entity.New},
		{Key: "status", Value: auction_entity.Active},
		{Key: "timestamp", Value: time.Now().Add(-1 * time.Hour).Unix()},
	}

	_, err = repo.Collection.InsertOne(ctx, auctionDoc)
	if err != nil {
		t.Fatal(err)
	}

	// Configurar duração do leilão para 0 minutos (para forçar expiração)
	os.Setenv("AUCTION_DURATION_MINUTES", "0")
	defer os.Unsetenv("AUCTION_DURATION_MINUTES")

	// Iniciar o monitor
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	repo.StartAuctionMonitor(ctx)

	// Aguardar o processamento
	time.Sleep(2 * time.Second)

	// Verificar se o leilão foi fechado
	var result bson.M
	err = repo.Collection.FindOne(ctx, bson.M{"_id": "test-auction-1"}).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, auction_entity.Completed, auction_entity.AuctionStatus(result["status"].(int32)))
}
