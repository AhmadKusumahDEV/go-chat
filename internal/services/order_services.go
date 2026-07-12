package services

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
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
	GetTransactionList(ctx context.Context, userId string) ([]*response.OrderResponse, error)
	MidtransNotification(ctx context.Context, notif *response.MidtransNotification) error
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
func (o *OrderServicesImpl) AddOrder(ctx context.Context, userId string) (*response.SnapResponse, error) {
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
		Plan:      "Subscription Premium",
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
func (o *OrderServicesImpl) GetTransactionList(ctx context.Context, userId string) ([]*response.OrderResponse, error) {
	transaction, err := o.OrderRepository.GetOrderByUserId(ctx, userId)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var result []*response.OrderResponse
	for _, v := range transaction {
		if v.Status == "pending" && time.Now().After(v.ExpiretAt) {
			v.Status = "expired"
		}

		resp := &response.OrderResponse{
			OrderId:   v.OrderID,
			Plan:      v.Plan,
			Amount:    v.Amount,
			Status:    v.Status,
			ExpiretAt: v.ExpiretAt,
			Username:  v.Username,
			Email:     v.Email,
		}

		if v.Status == "pending" && v.SnapToken.Valid {
			resp.Redirecturl = GenerateUrlRedirect(&o.cfg, v.SnapToken.String)
		}

		if v.PaymentMethod.Valid {
			resp.PaymentType = v.PaymentMethod.String
		}
		result = append(result, resp)
	}

	return result, nil
}

// MidtransNotification implements [OrderServices].
func (o *OrderServicesImpl) MidtransNotification(ctx context.Context, notif *response.MidtransNotification) error {
	if !SignatureValidate(notif.SignatureKey, notif.OrderID, notif.GrossAmount, notif.StatusCode, o.cfg.Payment.Midtrans.ServerKey) {
		return models.ErrSiganature
	}

	finalStatus := models.MapMidtransStatus(notif.TransactionStatus, notif.FraudStatus)
	if finalStatus == models.OrderStatusPending {
		log.Printf("Status masih pending untuk Order %s", notif.OrderID)
		return nil
	}

	payload, err := json.Marshal(notif)
	if err != nil {
		log.Println(err)
		return errors.New("internal server error")
	}

	err = o.OrderRepository.UpdatedOrder(ctx, &models.Order{
		OrderID:        notif.OrderID,
		WebHookPayload: payload,
		GatewayTxID: sql.NullString{
			String: notif.TransactionID,
			Valid:  true,
		},
		PaymentMethod: sql.NullString{
			String: notif.PaymentType,
			Valid:  true,
		},
		Status: finalStatus,
	})
	if err != nil {
		if errors.Is(err, models.ErrStatusAlreadySettled) {
			log.Println("Webhook Duplikat (Status sama persis)")
			return nil
		}

		if errors.Is(err, models.ErrSameStatus) {
			log.Println("duplicate webhook")
			return nil
		}
		return err
	}

	if finalStatus == models.OrderStatusSettled {
		// do somthing
		return nil
	}

	return nil
}

func SignatureValidate(signature, orderId, grossAmount, statusCode, serverKey string) bool {
	payload := orderId + statusCode + grossAmount + serverKey
	h := sha512.New()
	h.Write([]byte(payload))
	GenerateSignature := hex.EncodeToString(h.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(GenerateSignature), []byte(signature)) == 1
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
			Price:        90000,
			Quantity:     1,
			Name:         "Upgrade premium",
			Category:     "upgrade",
			MerchantName: "Atelier",
		},
		{
			ID:       "tax-1",
			Price:    9000,
			Quantity: 1,
			Name:     "tax 10%",
		},
	}

	// grossAmount := countGrossAmount(listItem)

	return request.SnapRequest{
		TransactionInfo: models.TransactionDetail{
			OrderID:     orderId,
			GrossAmount: 99000,
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
	return &OrderServicesImpl{
		cfg:             cfg,
		userRepository:  userRepository,
		httpClient:      httpClient,
		OrderRepository: orderRepository,
		rds:             redis,
	}
}
