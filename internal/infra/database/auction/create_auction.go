package auction

import (
	"context"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}

type AuctionRepository struct {
	Collection *mongo.Collection
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection: database.Collection("auctions"),
	}
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	return nil
}

func (ar *AuctionRepository) UpdateAuctionStatus(
	ctx context.Context,
	auctionId string,
	status auction_entity.AuctionStatus) *internal_error.InternalError {

	filter := bson.M{"_id": auctionId}
	update := bson.M{"$set": bson.M{"status": status}}

	_, err := ar.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.Error("Error trying to update auction status", err)
		return internal_error.NewInternalServerError("Error trying to update auction status")
	}

	return nil
}

func (ar *AuctionRepository) FindExpiredAuctions(
	ctx context.Context) ([]string, *internal_error.InternalError) {

	auctionDuration, err := strconv.Atoi(os.Getenv("AUCTION_DURATION_MINUTES"))
	if err != nil {
		auctionDuration = 60 // valor padrão de 60 minutos
	}

	expirationTime := time.Now().Add(-time.Duration(auctionDuration) * time.Minute)

	filter := bson.M{
		"status": auction_entity.Active,
		"timestamp": bson.M{
			"$lt": expirationTime.Unix(),
		},
	}

	cursor, err := ar.Collection.Find(ctx, filter)
	if err != nil {
		logger.Error("Error trying to find expired auctions", err)
		return nil, internal_error.NewInternalServerError("Error trying to find expired auctions")
	}
	defer cursor.Close(ctx)

	var auctions []AuctionEntityMongo
	if err = cursor.All(ctx, &auctions); err != nil {
		logger.Error("Error trying to decode expired auctions", err)
		return nil, internal_error.NewInternalServerError("Error trying to decode expired auctions")
	}

	var auctionIds []string
	for _, auction := range auctions {
		auctionIds = append(auctionIds, auction.Id)
	}

	return auctionIds, nil
}

func (ar *AuctionRepository) StartAuctionMonitor(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Obter a duração do leilão das variáveis de ambiente
				auctionDuration, err := strconv.Atoi(os.Getenv("AUCTION_DURATION_MINUTES"))
				if err != nil {
					auctionDuration = 60 // valor padrão de 60 minutos
				}

				// Calcular o tempo limite
				expirationTime := time.Now().Add(-time.Duration(auctionDuration) * time.Minute)

				// Buscar e atualizar leilões expirados
				filter := bson.M{
					"status": auction_entity.Active,
					"timestamp": bson.M{
						"$lt": expirationTime.Unix(),
					},
				}

				update := bson.M{
					"$set": bson.M{
						"status": auction_entity.Completed,
					},
				}

				_, err = ar.Collection.UpdateMany(ctx, filter, update)
				if err != nil {
					logger.Error("Error updating expired auctions", err)
				}
			}
		}
	}()
}
