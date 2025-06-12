package firebaseClient

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
	"log"
	"os"
)

func ProvideFirebaseApp() *firebase.App {
	ctx := context.Background()

	// Read file path from env or fallback
	credFile := os.Getenv("FIREBASE_CREDENTIALS_FILE")
	if credFile == "" {
		log.Fatal("FIREBASE_CREDENTIALS_FILE not set")
	}

	log.Printf("invoke firebaseClient with credentials file: %s", credFile)

	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: "swptest-7f1bb",
	}, option.WithCredentialsFile(credFile))
	if err != nil {
		log.Fatalf("error initializing firebaseClient app: %v", err)
	}

	return app
}
