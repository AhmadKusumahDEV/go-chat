package config

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FirebaseConfig struct {
	Path string `mapstructure:"credential_path"`
}

// Inisialisasi Firebase App
func InitFirebase(ctx context.Context, projectID string, credentialPath string) (*firebase.App, error) {
	if projectID == "" {
		return nil, fmt.Errorf("firebase project id is required")
	}

	if credentialPath == "" {
		return nil, fmt.Errorf("firebase credential path is required")
	}

	opt := option.WithCredentialsFile(credentialPath)

	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: projectID,
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase app: %w", err)
	}

	return app, nil
}

// Kirim notifikasi ke satu device
func SendToDevice(ctx context.Context, client *messaging.Client, token string) error {
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: "Hello from Firebase Message Cloud!",
			Body:  "This is a test notification",
		},
		Token: token, // Device FCM token
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	fmt.Println("Successfully sent message:", response)
	return nil
}

// Kirim ke multiple devices
func SendToMultipleDevices(ctx context.Context, client *messaging.Client, tokens []string) error {
	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: "Broadcast Message",
			Body:  "Sent to multiple devices",
		},
		Tokens: tokens,
	}

	response, err := client.SendMulticast(ctx, message)
	if err != nil {
		return fmt.Errorf("error sending multicast: %v", err)
	}

	fmt.Printf("Successfully sent %d messages\n", response.SuccessCount)
	if response.FailureCount > 0 {
		fmt.Printf("Failed to send %d messages\n", response.FailureCount)
	}
	return nil
}
