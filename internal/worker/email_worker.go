package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/queue"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/rabbitmq/amqp091-go"
)

const (
	MaxRetries    = 3
	RetryDelayMs  = 5000 // 5 seconds
	RetryHeader   = "x-retry-count"
	OriginalRoute = "x-original-routing-key"
	StepHeader    = "x-failed-step"
)

const (
	StepGetOrder   = "get_order"
	StepUpdateTier = "update_tier"
	StepSendEmail  = "send_email"
)

var (
	EmailOtp            = "otp"
	EmailPaymentSuccess = "payment_success"
)

type RetryableError struct {
	Step     string
	Err      error
	CanRetry bool
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("step %s failed: %v (retryable: %v)", e.Step, e.Err, e.CanRetry)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

type EmailWorker struct {
	rabbitmq   *amqp091.Channel
	httpClient *http.Client
	espConfig  config.Esp
	order      repository.OrderRepository
	user       repository.RepositoryUser
}

func NewEmailWorker(
	rabbitmq *amqp091.Channel,
	httpClient *http.Client,
	usersRepository repository.RepositoryUser,
	orderRepository repository.OrderRepository,
	espConfig config.Esp,
) *EmailWorker {
	return &EmailWorker{
		rabbitmq:   rabbitmq,
		httpClient: httpClient,
		espConfig:  espConfig,
		order:      orderRepository,
		user:       usersRepository,
	}
}

func (w *EmailWorker) Start(ctx context.Context) error {
	msgs, err := w.rabbitmq.Consume(
		"email-notifications",
		"email-worker",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	log.Println("EmailWorker started, waiting for messages...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("EmailWorker shutting down...")
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				w.processEmail(&msg)
			}
		}
	}()

	return nil
}

// processEmail handles incoming email messages with retry logic
func (w *EmailWorker) processEmail(msg *amqp091.Delivery) {
	log.Printf("Processing email event: %s", msg.MessageId)
	ctx := context.Background()

	// Get retry count
	retryCount := w.getRetryCount(msg)

	// Unmarshal the event
	var event queue.EmailEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("Failed to unmarshal email event: %v", err)
		log.Printf("   Raw body: %s", string(msg.Body))
		msg.Nack(false, false) // Send to DLQ - malformed message
		return
	}

	log.Printf("Email to: %s | Subject: %s | Type: %s | Retry: %d",
		event.To, event.Subject, event.Type, retryCount)

	switch event.Type {
	case EmailOtp:
		w.processOtpEmail(ctx, msg, &event)

	case EmailPaymentSuccess:
		w.processPaymentSuccessEmail(ctx, msg, &event, retryCount)

	default:
		log.Printf("Unknown email type: %s", event.Type)
		msg.Ack(false)
	}
}

func (w *EmailWorker) processOtpEmail(ctx context.Context, msg *amqp091.Delivery, event *queue.EmailEvent) {
	payload := w.buildOTPEmail(event.To, event.OTP)
	if payload == nil {
		log.Printf("Failed to build OTP email payload")
		msg.Nack(false, false) // Don't retry malformed content
		return
	}

	if err := w.sendEmail(payload); err != nil {
		log.Printf("Failed to send OTP email: %v", err)
		w.handleRetry(ctx, msg, event, StepSendEmail, err, false)
		return
	}

	log.Printf("OTP email sent successfully to: %s", event.To)
	msg.Ack(false)
}

func (w *EmailWorker) processPaymentSuccessEmail(ctx context.Context, msg *amqp091.Delivery, event *queue.EmailEvent, retryCount int) {
	order, err := w.order.GetOrderByOrderID(ctx, event.OrderID)
	if err != nil {
		log.Printf("Failed to get order details: %v", err)
		w.handleRetry(ctx, msg, event, StepGetOrder, err, true)
		return
	}

	log.Printf("Order retrieved: %s (User: %s, Email: %s)",
		order.OrderID, order.UserID.String(), order.Email)

	if err := w.user.UpdateTierUser(ctx, order.UserID.String(), "premium"); err != nil {
		log.Printf("Failed to update user tier: %v", err)
		w.handleRetry(ctx, msg, event, StepUpdateTier, err, true)
		return
	}

	log.Printf("User tier updated to premium for: %s", order.UserID.String())

	payload := w.buildPaymentSuccessEmail(order.Email, order.Username, order.Plan, order.OrderID, order.Amount)
	if payload == nil {
		log.Printf("Step 3 - Failed to build payment email payload")
		w.handleRetry(ctx, msg, event, StepSendEmail, fmt.Errorf("failed to build email payload"), false)
		return
	}

	if err := w.sendEmail(payload); err != nil {
		log.Printf("Step 3 - Failed to send payment email: %v", err)
		w.handleRetry(ctx, msg, event, StepSendEmail, err, false)
		return
	}

	log.Printf("Step 3 - Payment email sent successfully to: %s", order.Email)

	log.Printf("Payment success email flow completed for order: %s", order.OrderID)
	msg.Ack(false)
}

func (w *EmailWorker) handleRetry(ctx context.Context, msg *amqp091.Delivery, event *queue.EmailEvent, failedStep string, err error, canRetry bool) {
	retryCount := w.getRetryCount(msg)

	if !canRetry || retryCount >= MaxRetries {
		if retryCount >= MaxRetries {
			log.Printf("Max retries (%d) exceeded for %s, sending to DLQ", MaxRetries, failedStep)
		} else {
			log.Printf("Error in %s is not retryable, sending to DLQ", failedStep)
		}
		msg.Nack(false, false)
		return
	}

	newRetryCount := retryCount + 1
	log.Printf("🔄 Retrying %s (attempt %d/%d)", failedStep, newRetryCount, MaxRetries)

	delayExchange := "delay.exchange"

	// Build retry message
	retryEvent := &queue.EmailEvent{
		Type:     event.Type,
		To:       event.To,
		Subject:  event.Subject,
		OTP:      event.OTP,
		Username: event.Username,
		PlanName: event.PlanName,
		OrderID:  event.OrderID,
		Amount:   event.Amount,
	}

	body, _ := json.Marshal(retryEvent)

	headers := amqp091.Table{
		RetryHeader:   int32(newRetryCount),
		StepHeader:    failedStep,
		OriginalRoute: msg.RoutingKey,
	}

	err = w.rabbitmq.PublishWithContext(
		ctx,
		delayExchange,
		"payment.retry",
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent,
			Headers:      headers,
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		log.Printf("Failed to publish retry message: %v", err)
		msg.Nack(false, true)
		return
	}

	log.Printf("Retry message published to delay queue, will be retried in %dms", RetryDelayMs)

	msg.Ack(false)
}

func (w *EmailWorker) getRetryCount(msg *amqp091.Delivery) int {
	if msg.Headers == nil {
		return 0
	}

	if count, ok := msg.Headers[RetryHeader]; ok {
		switch v := count.(type) {
		case int32:
			return int(v)
		case int64:
			return int(v)
		case int:
			return v
		}
	}

	return 0
}

// sendEmail sends an email via Brevo API
func (w *EmailWorker) sendEmail(payload []byte) error {
	req, err := http.NewRequest("POST", w.espConfig.Brevo.Url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", w.espConfig.Brevo.ApiKey)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return &RetryableError{
			Step:     StepSendEmail,
			Err:      fmt.Errorf("Brevo API error: status %d", resp.StatusCode),
			CanRetry: true,
		}
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Brevo API error: status %d", resp.StatusCode)
	}

	return nil
}

// buildOTPEmail creates the OTP verification email HTML
func (w *EmailWorker) buildOTPEmail(recipientEmail, otp string) []byte {
	htmlContent := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<body style="background-color: #f4f5f7; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; padding: 40px 0;">
			<div style="max-width: 600px; margin: 0 auto; background-color: #ffffff; padding: 40px; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.05);">
				<div style="text-align: center; margin-bottom: 30px;">
					<h2 style="color: #333333; font-size: 24px; margin: 0;">Verifikasi Akun</h2>
				</div>
				<p style="color: #555555; font-size: 16px; line-height: 1.5;">Halo,</p>
				<p style="color: #555555; font-size: 16px; line-height: 1.5;">Kami menerima permintaan untuk melakukan aktivitas pada akun Anda. Silakan masukkan kode verifikasi berikut ke dalam aplikasi:</p>
				<div style="text-align: center; margin: 35px 0;">
					<div style="display: inline-block; font-size: 36px; font-weight: bold; color: #1a73e8; letter-spacing: 8px; padding: 15px 30px; background-color: #f8f9fa; border: 1px dashed #c6d4e1; border-radius: 6px;">
						%s
					</div>
				</div>
				<p style="color: #555555; font-size: 14px; line-height: 1.5;">Kode ini hanya berlaku selama <b>5 menit</b>. Jangan bagikan kode ini kepada siapa pun, termasuk pihak admin.</p>
				<hr style="border: none; border-top: 1px solid #eeeeee; margin: 30px 0;" />
				<p style="color: #999999; font-size: 12px; text-align: center; line-height: 1.5;">
					Jika Anda tidak merasa melakukan permintaan ini, abaikan email ini.<br>
					&copy; 2026 Madgo. Semua hak dilindungi.
				</p>
			</div>
		</body>
		</html>
	`, otp)

	return w.createEmailPayload(recipientEmail, "Kode Verifikasi Madgo Anda", htmlContent)
}

// buildPaymentSuccessEmail creates the payment success notification email HTML
func (w *EmailWorker) buildPaymentSuccessEmail(recipientEmail, username, planName, orderID string, amount int64) []byte {
	formattedAmount := fmt.Sprintf("Rp %d", amount)

	htmlContent := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<body style="background-color: #f4f5f7; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; padding: 40px 0;">
			<div style="max-width: 600px; margin: 0 auto; background-color: #ffffff; padding: 40px; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.05);">
				<div style="text-align: center; margin-bottom: 30px;">
					<div style="display: inline-block; width: 60px; height: 60px; background-color: #4CAF50; border-radius: 50%%; margin-bottom: 20px; line-height: 60px;">
						<span style="color: white; font-size: 30px;">✓</span>
					</div>
					<h2 style="color: #333333; font-size: 24px; margin: 0;">Pembayaran Berhasil!</h2>
				</div>
				<p style="color: #555555; font-size: 16px; line-height: 1.5;">Halo <strong>%s</strong>,</p>
				<p style="color: #555555; font-size: 16px; line-height: 1.5;">Selamat! Pembayaran Anda telah berhasil diproses. Berikut adalah detail transaksi Anda:</p>
				<div style="background-color: #f8f9fa; border-radius: 8px; padding: 20px; margin: 25px 0;">
					<table style="width: 100%%; border-collapse: collapse;">
						<tr>
							<td style="color: #666666; padding: 8px 0; border-bottom: 1px solid #eeeeee;">Order ID</td>
							<td style="color: #333333; font-weight: 600; text-align: right; padding: 8px 0; border-bottom: 1px solid #eeeeee;">%s</td>
						</tr>
						<tr>
							<td style="color: #666666; padding: 8px 0; border-bottom: 1px solid #eeeeee;">Paket</td>
							<td style="color: #333333; font-weight: 600; text-align: right; padding: 8px 0; border-bottom: 1px solid #eeeeee;">%s</td>
						</tr>
						<tr>
							<td style="color: #666666; padding: 8px 0;">Total Bayar</td>
							<td style="color: #4CAF50; font-weight: 700; font-size: 18px; text-align: right; padding: 8px 0;">%s</td>
						</tr>
					</table>
				</div>
				<p style="color: #555555; font-size: 16px; line-height: 1.5;">Akun Anda sekarang telah diupgrade ke <strong>Premium</strong>. Nikmati semua fitur eksklusif yang tersedia!</p>
				<div style="text-align: center; margin: 30px 0;">
					<a href="#" style="display: inline-block; background-color: #1a73e8; color: #ffffff; text-decoration: none; padding: 14px 28px; border-radius: 6px; font-weight: 600;">Mulai Gunakan Premium</a>
				</div>
				<hr style="border: none; border-top: 1px solid #eeeeee; margin: 30px 0;" />
				<p style="color: #999999; font-size: 12px; text-align: center; line-height: 1.5;">
					Jika Anda memiliki pertanyaan, jangan ragu untuk menghubungi tim support kami.<br>
					&copy; 2026 Madgo. Semua hak dilindungi.
				</p>
			</div>
		</body>
		</html>
	`, username, orderID, planName, formattedAmount)

	return w.createEmailPayload(recipientEmail, "Pembayaran Berhasil - Selamat Menggunakan Premium!", htmlContent)
}

// createEmailPayload creates the Brevo API payload
func (w *EmailWorker) createEmailPayload(recipientEmail, subject, htmlContent string) []byte {
	payload := map[string]interface{}{
		"sender": map[string]string{
			"name":  "madgov",
			"email": "noreply@madgo.my.id",
		},
		"to": []map[string]string{
			{
				"email": recipientEmail,
			},
		},
		"subject":     subject,
		"htmlContent": htmlContent,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal email payload: %v", err)
		return nil
	}

	return jsonPayload
}
