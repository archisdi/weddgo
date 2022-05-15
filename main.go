package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	firebase "firebase.google.com/go/v4"
	"github.com/gosimple/slug"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// TODO: seperate 3rd party into singleton modules
// TODO: implement goroutine to update invitation link
// TODO: simplify error logging

type Invitation struct {
	Name     string `json:"name"`
	Domicile string `json:"domicile"`
	Priority int    `json:"priority"`
	Invitee  string `json:"invitee"`
	Gender   string `json:"gender"`
	Prefix   string `json:"prefix"`
	Key      string `json:"key"`
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

	regenerateLink := os.Getenv("REGENERATE_LINK") == "1"
	invitations := []Invitation{}
	for i, row := range sheet.Values {
		priority, _ := strconv.Atoi(row[2].(string))

		name := row[0].(string)
		key := slug.Make(name)

		if regenerateLink {
			index := i + 2
			col := strconv.Itoa(index)
			var vr sheets.ValueRange
			myval := []interface{}{os.Getenv("INVITATION_BASE_URL") + "/" + key}
			vr.Values = append(vr.Values, myval)
			sheetsService.Spreadsheets.Values.Update(sheetID, os.Getenv("LINK_SHEET_COL")+col, &vr).ValueInputOption("RAW").Do()
		}

		var prefix string
		if len(row) > 5 {
			prefix = row[5].(string)
		}
		invitations = append(invitations, Invitation{
			Name:     name,
			Domicile: row[1].(string),
			Priority: priority,
			Invitee:  row[3].(string),
			Gender:   row[4].(string),
			Prefix:   prefix,
			Key:      key,
		})
	}

	invitationMap := make(InvitationMap)
	for _, inv := range invitations {
		invitationMap[inv.Key] = inv
	}

	dbClient, err := firebase.Database(ctx)
	if err != nil {
		log.Fatalln("Error getting database client :", err)
		os.Exit(1)
	}

	ref := dbClient.NewRef("invitation/guest")
	err = ref.Set(ctx, &invitationMap)
	if err != nil {
		log.Fatalln("Error updating database :", err)
		os.Exit(1)
	}

	jsonFile, err := os.Open(os.Getenv("DETAIL_FILE_URL"))
	if err != nil {
		log.Fatalln("Error opening json file :", err)
		os.Exit(1)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var detail map[string]interface{}
	json.Unmarshal([]byte(byteValue), &detail)

	detailRef := dbClient.NewRef("invitation/digital")
	err = detailRef.Set(ctx, &detail)
	if err != nil {
		log.Fatalln("Error updating database :", err)
		os.Exit(1)
	}
}
