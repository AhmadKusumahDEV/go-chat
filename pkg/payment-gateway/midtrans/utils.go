package midtrans

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func GenerateOrderID(now time.Time) string {
	date := now.Format("20060102")
	random := RandomString(8)
	return fmt.Sprintf("INV-%s-%s", date, random)
}

func RandomString(n int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func GenerateTimeZone() (string, time.Time) {
	now := time.Now()

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Println("error when load location", err)
		loc = time.FixedZone("WIB", 7*3600)
	}

	startTimeStr := now.In(loc).Format("2006-01-02 15:04:05 -0700")
	expiredAtDB := now.Add(time.Hour * 1).UTC()

	return startTimeStr, expiredAtDB
}

// validate signature only for midtrans
func SignatureValidate(signature, orderId, grossAmount, statusCode, serverKey string) bool {
	payload := orderId + statusCode + grossAmount + serverKey
	h := sha512.New()
	h.Write([]byte(payload))
	GenerateSignature := hex.EncodeToString(h.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(GenerateSignature), []byte(signature)) == 1
}

func GenerateUrlRedirect(cfg Midtrans, snapToken string) string {
	if cfg.Mode != "production" {
		return fmt.Sprintf(cfg.UrlRedirectSanbox, snapToken)
	}
	return fmt.Sprintf(cfg.UrlRedirectProduction, snapToken)
}

func GenerateUrlSnap(cfg Midtrans) string {
	if cfg.Mode != "production" {
		return cfg.UrlSnapSanbox
	}
	return cfg.UrlSnapProduction
}

func CreateSnapToken(ctx context.Context, httpClient *http.Client, data []byte, cfg Midtrans) (*SnapResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", GenerateUrlSnap(cfg), bytes.NewReader(data))
	if err != nil {
		log.Println(err)
		return nil, errors.New("error when create request")
	}

	auth := base64.StdEncoding.EncodeToString([]byte(cfg.ServerKey + ":"))
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(httpReq)
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

	var result SnapResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse midtrans response: %w", err)
	}

	return &result, nil
}

func CountGrossAmount(item []MerchantItemDetail) int64 {
	var total int64
	for _, v := range item {
		total += v.Price * int64(v.Quantity)
	}
	return total
}

func MappingChargePaymentType(method string, charge *ChargeRequest) {

	switch method {
	case "gopay":
		charge.GoPay = &GoPayRequest{
			EnableCallback: true,                                            // Wajib true jika ingin user diredirect setelah bayar di app Gojek
			CallbackURL:    "https://domainkamu.com/payment/gopay/callback", // URL frontend/deeplink kamu
		}

	case "shopeepay":
		charge.ShopeePay = &ShopeePayRequest{
			CallbackUrl: "https://domainkamu.com/payment/shopeepay/callback", // Wajib
		}

	case "dana", "linkaja":

	case "bca_va", "bni_va", "bri_va", "permata_va":
		bankName := method[:3]
		if method == "permata_va" {
			bankName = "permata"
		}
		charge.PaymentType = "bank_transfer"
		charge.BankTransfer = &BankTransferRequest{
			Bank: bankName,
		}

	case "mandiri_va", "echannel":
		charge.PaymentType = "echannel"
		charge.EChannel = &EChannelDetail{
			BillInfo1: "Pembayaran",
			BillInfo2: "Pesanan",
		}

	case "indomaret":
		charge.PaymentType = "cstore"
		charge.CStore = &CStoreRequest{
			Store:   method,
			Message: "Pembayaran Transaksi",
		}

	case "alfamart":
		charge.PaymentType = "cstore"
		charge.CStore = &CStoreRequest{
			Store:             method,
			Message:           "pembayaran Transaksi",
			AlfamartFreeText1: "terimaksih sudah berbelanja",
			AlfamartFreeText2: "di toko kami",
			AlfamartFreeText3: "CS/Bantuan: 0812-3456-7890",
		}

	case "qris":
		charge.PaymentType = "qris"
		charge.QRIS = &QRISRequest{
			Acquirer: "gopay",
		}

	default:
		log.Printf("Warning: unknown payment method %s", method)
	}
}

// NewChargeRequest creates a new charge request with transaction details
func NewChargeRequest(orderID string, grossAmount int64, paymentType string) *ChargeRequest {
	return &ChargeRequest{
		PaymentType: paymentType,
		TransactionDetails: TransactionDetails{
			OrderID:     orderID,
			GrossAmount: grossAmount,
		},
	}
}

func (c *ChargeRequest) SetCustomerDetails(firstName, lastName, email, phone string) {
	c.CustomerDetails = &CustomerDetails{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Phone:     phone,
	}
}

func (c *ChargeRequest) SetCreditCardToken(tokenID string, saveToken bool) {
	c.CreditCard = &CreditCardRequest{
		TokenID:        tokenID,
		Authentication: true, // Enable 3D secure by default
	}
}

func (c *ChargeRequest) SetGoPayOptions(callbackURL string, promotionIDs []string) {
	if c.GoPay == nil {
		c.GoPay = &GoPayRequest{}
	}
	c.GoPay.CallbackURL = callbackURL
	c.GoPay.EnableCallback = callbackURL != ""
	if len(promotionIDs) > 0 {
		c.GoPay.PromotionIDs = promotionIDs
	}
}

func (c *ChargeRequest) SetBankTransferOptions(bank, vaNumber, recipientName string) {
	if c.BankTransfer == nil {
		c.BankTransfer = &BankTransferRequest{}
	}
	c.BankTransfer.Bank = bank
	c.BankTransfer.VANumber = vaNumber
	c.BankTransfer.RecipientName = recipientName
}

func (c *ChargeRequest) SetCStoreOptions(message string) {
	if c.CStore == nil {
		return
	}
	c.CStore.Message = message
}

func (c *ChargeRequest) SetQRISOptions(acquirer string) {
	if c.QRIS == nil {
		c.QRIS = &QRISRequest{}
	}
	c.QRIS.Acquirer = acquirer
}

func IsAvailableRefund(paymentType string) bool {
	switch paymentType {
	case "gopay", "shopeepay", "dana", "linkaja", "ovo":
		return true

	default:
		return false
	}
}
