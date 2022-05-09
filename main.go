package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	firebase "firebase.google.com/go/v4"
	"github.com/gosimple/slug"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Invitation struct {
	Name     string `json:"name"`
	Domicile string `json:"domicile"`
	Priority int    `json:"priority"`
	Invitee  string `json:"invitee"`
}

type InvitationMap map[string]Invitation

func getFirebaseAppInstance() *firebase.App {
	conf := &firebase.Config{
		DatabaseURL: os.Getenv("FIREBASE_DB_URL"),
	}
	// Fetch the service account key JSON file contents
	opt := option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))

	// Initialize the app with a service account, granting admin privileges
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, conf, opt)
	if err != nil {
		log.Fatalln("Error initializing app:", err)
		os.Exit(1)
	}

	return app
}

func main() {
	if envErr := godotenv.Load(); envErr != nil {
		log.Fatal("error while loading environment file")
		os.Exit(1)
	}

	sheetID := os.Getenv("SHEET_ID")

	ctx := context.Background()
	sheetsService, err := sheets.NewService(ctx)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	firebase := getFirebaseAppInstance()

	readRange := os.Getenv("SHEET_RANGE")
	sheet, err := sheetsService.Spreadsheets.Values.Get(sheetID, readRange).Do()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	invitations := []Invitation{}
	for _, row := range sheet.Values {
		priority, _ := strconv.Atoi(row[2].(string))
		invitations = append(invitations, Invitation{
			Name:     row[0].(string),
			Domicile: row[1].(string),
			Priority: priority,
			Invitee:  row[3].(string),
		})
	}

	invitationMap := make(InvitationMap)
	for _, inv := range invitations {
		mapKey := slug.Make(inv.Name)
		invitationMap[mapKey] = inv
	}

	dbClient, err := firebase.Database(ctx)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	ref := dbClient.NewRef("invitation/public")
	err = ref.Set(ctx, &invitationMap)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
