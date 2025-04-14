package main

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/infra/database/auction"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Conectar ao MongoDB
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://admin:admin@localhost:27017/auctions?authSource=admin"))
	if err != nil {
		log.Fatal("Erro ao conectar ao MongoDB:", err)
	}
	defer client.Disconnect(ctx)

	// Criar repositório
	repo := auction.NewAuctionRepository(client.Database("auctions"))

	// Criar um leilão expirado
	auction, internalErr := auction_entity.CreateAuction(
		"Produto Teste",
		"Cat",
		"Descrição do produto teste com mais de 10 caracteres",
		auction_entity.New,
	)
	if internalErr != nil {
		log.Fatal("Erro ao criar leilão:", internalErr.Message)
	}

	// Definir timestamp para 1 hora atrás
	auction.Timestamp = time.Now().Add(-1 * time.Hour)

	// Inserir o leilão
	internalErr = repo.CreateAuction(ctx, auction)
	if internalErr != nil {
		log.Fatal("Erro ao inserir leilão:", internalErr.Message)
	}

	log.Printf("Leilão criado com ID: %s\n", auction.Id)
	log.Println("Status inicial:", auction.Status)

	// Iniciar o monitor
	repo.StartAuctionMonitor(ctx)

	// Aguardar 5 segundos para o monitor processar
	time.Sleep(5 * time.Second)

	// Verificar o status do leilão
	foundAuction, internalErr := repo.FindAuctionById(ctx, auction.Id)
	if internalErr != nil {
		log.Fatal("Erro ao buscar leilão:", internalErr.Message)
	}

	log.Println("Status após monitor:", foundAuction.Status)
}
