package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/redis/go-redis/v9"
)

type OrderServices interface {
	GetSnapToken(userId string) (string, error)
	MidtransNotification(req map[string]interface{})
	AddOrder(ctx context.Context, userId string) (*response.SnapResponse, error)
}

type OrderServicesImpl struct {
	cfg             config.Cfg
	userRepository  repository.RepositoryUser
	OrderRepository repository.OrderRepository
	httpClient      *http.Client
	rds             *redis.Client
}

// AddOrder implements [OrderServices].
func (o OrderServicesImpl) AddOrder(ctx context.Context, userId string) (*response.SnapResponse, error) {
	keyLock := fmt.Sprintf("create_order:lock:%s", userId)

	lockStat, err := o.rds.SetNX(ctx, keyLock, "1", 10*time.Second).Result()
	if err != nil || !lockStat {
		log.Println(err)
		return nil, errors.New("transaksi sedang di proses mohon meunggu")
	}

	token, err := o.OrderRepository.GetActiveOrder(ctx, userId)
	if err == nil && token.Valid {
		return &response.SnapResponse{
			Token:       token.String,
			RedirectURL: GenerateUrlRedirect(&o.cfg, token.String),
		}, nil
	}

	defer o.rds.Del(context.Background(), keyLock)

	user, err := o.userRepository.FindByID(ctx, userId)
	if err != nil {
		return &response.SnapResponse{}, errors.New("User Tidak ditemukan")
	}

	orderId := models.GenerateOrderID(time.Now())
	req, expiredDb := CreateDataTransaction(orderId, user, &o.cfg)

	payload, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", GenerateUrlSnap(&o.cfg), bytes.NewReader(payload)) //http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		log.Println(err)
		return nil, errors.New("error when create request")
	}

	auth := base64.StdEncoding.EncodeToString([]byte(o.cfg.Payment.Midtrans.ServerKey + ":"))
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		log.Println(err)
		return nil, errors.New("got error when request to api")
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		log.Printf("midtrans error [%d]: %s", resp.StatusCode, string(respBody))
		return nil, errors.New("error status receiver")
	}

	var result response.SnapResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse midtrans response: %w", err)
	}

	err = o.OrderRepository.CreateOrder(ctx, &models.Order{
		OrderID:   orderId,
		UserID:    user.ID,
		Plan:      "premium",
		Amount:    req.TransactionInfo.GrossAmount,
		Status:    "pending",
		Gateway:   "midtrans",
		ExpiretAt: expiredDb,
		SnapToken: sql.NullString{
			String: result.Token,
			Valid:  true,
		},
	})
	if err != nil {
		log.Println(err)
		return nil, errors.New("errors when store db")
	}

	return &result, nil
}

// GetSnapToken implements [OrderServices].
func (o OrderServicesImpl) GetSnapToken(userId string) (string, error) {
	panic("unimplemented")
}

// MidtransNotification implements [OrderServices].
func (o OrderServicesImpl) MidtransNotification(req map[string]interface{}) {
	panic("unimplemented")
}

func GenerateUrlRedirect(cfg *config.Cfg, snapToken string) string {
	if cfg.Payment.Midtrans.Mode != "production" {
		return fmt.Sprintf(cfg.Payment.Midtrans.UrlRedirectSanbox, snapToken)
	}
	return fmt.Sprintf(cfg.Payment.Midtrans.UrlRedirectProduction, snapToken)
}

func GenerateUrlSnap(cfg *config.Cfg) string {
	if cfg.Payment.Midtrans.Mode != "production" {
		return cfg.Payment.Midtrans.UrlSnapSanbox
	}
	return cfg.Payment.Midtrans.UrlSnapProduction
}

func CreateDataTransaction(orderId string, user *models.Users, cfg *config.Cfg) (request.SnapRequest, time.Time) {
	now := time.Now()

	loc, _ := time.LoadLocation("Asia/Jakarta")

	startTimeStr := now.In(loc).Format("2006-01-02 15:04:05 -0700")

	expiredAtDB := now.Add(time.Hour * 1).UTC()

	listItem := []models.MerchantItemDetail{
		{
			ID:           "PKG-PREMIUM",
			Price:        9000,
			Quantity:     1,
			Name:         "Upgrade user to premium",
			Category:     "upgrade",
			MerchantName: "Atelier",
		},
		{
			ID:       "tax-1",
			Price:    1000,
			Quantity: 1,
			Name:     "tax 10%",
		},
	}

	// grossAmount := countGrossAmount(listItem)

	return request.SnapRequest{
		TransactionInfo: models.TransactionDetail{
			OrderID:     orderId,
			GrossAmount: 10000,
		},
		ItemInfo: listItem,
		CustomerInfo: models.CustomerDetail{
			FirstName: user.Username,
			Email:     user.Email,
		},
		ListPaymentEnable: cfg.Payment.Midtrans.AllowMethodPayment,
		Expired: models.Expiry{
			StartTime: startTimeStr,
			Unit:      "hours",
			Duration:  1,
		},
		CustomField1: user.ID.String(),
	}, expiredAtDB
}

// use this func when have more than one item
func countGrossAmount(item []models.MerchantItemDetail) int64 {
	var total int64
	for _, v := range item {
		total += v.Price * int64(v.Quantity)
	}
	return total
}

func NewOrderServices(cfg config.Cfg, userRepository repository.RepositoryUser, httpClient *http.Client, orderRepository repository.OrderRepository, redis *redis.Client) OrderServices {
	return OrderServicesImpl{
		cfg:             cfg,
		userRepository:  userRepository,
		httpClient:      httpClient,
		OrderRepository: orderRepository,
		rds:             redis,
	}
}
