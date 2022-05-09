package main

import (
	"context"
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

func getFirebaseAppInstance(ctx context.Context) *firebase.App {
	conf := &firebase.Config{
		DatabaseURL: os.Getenv("FIREBASE_DB_URL"),
	}
	opt := option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	app, err := firebase.NewApp(ctx, conf, opt)
	if err != nil {
		log.Fatalln("Error initializing app:", err)
		os.Exit(1)
	}
	return app
}

func getSheetInstance(ctx context.Context) *sheets.Service {
	sheetsService, err := sheets.NewService(ctx)
	if err != nil {
		log.Fatalln("Error initializing sheet service:", err)
		os.Exit(1)
	}
	return sheetsService
}

func main() {
	if envErr := godotenv.Load(); envErr != nil {
		log.Fatal("error while loading environment file")
		os.Exit(1)
	}

	ctx := context.Background()
	sheetsService := getSheetInstance(ctx)
	firebase := getFirebaseAppInstance(ctx)

	sheetID := os.Getenv("SHEET_ID")
	readRange := os.Getenv("SHEET_RANGE")
	sheet, err := sheetsService.Spreadsheets.Values.Get(sheetID, readRange).Do()
	if err != nil {
		log.Fatalln("Error reading sheet :", err)
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
		log.Fatalln("Error getting database client :", err)
		os.Exit(1)
	}

	ref := dbClient.NewRef("invitation/public")
	err = ref.Set(ctx, &invitationMap)
	if err != nil {
		log.Fatalln("Error updating database :", err)
		os.Exit(1)
	}
}
