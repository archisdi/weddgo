package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"weddgo/modules"

	"github.com/gosimple/slug"
	"github.com/joho/godotenv"
	"google.golang.org/api/sheets/v4"
)

var SheetID string

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

func regenerateInvivationLink(invitations []Invitation) {
	var vr sheets.ValueRange
	for _, invitation := range invitations {
		vr.Values = append(vr.Values, []interface{}{os.Getenv("INVITATION_BASE_URL") + "/" + invitation.Key})
	}
	modules.SH.Spreadsheets.Values.Update(SheetID, os.Getenv("LINK_SHEET_COL"), &vr).ValueInputOption("RAW").Do()
}

func main() {
	if envErr := godotenv.Load(); envErr != nil {
		log.Fatal("error while loading environment file")
		os.Exit(1)
	}

	ctx := context.Background()
	modules.InitializeFirebase(ctx)
	modules.InitializeSheet(ctx)

	SheetID = os.Getenv("SHEET_ID")
	readRange := os.Getenv("SHEET_RANGE")
	sheet, err := modules.SH.Spreadsheets.Values.Get(SheetID, readRange).Do()
	if err != nil {
		log.Fatalln("Error reading sheet :", err)
		os.Exit(1)
	}

	regenerateLink := os.Getenv("REGENERATE_LINK") == "1"
	invitations := []Invitation{}

	for _, row := range sheet.Values {
		id := row[0].(string)
		priority, _ := strconv.Atoi(row[3].(string))
		sheetKey := slug.Make(id)

		var prefix string
		if len(row) > 6 {
			prefix = row[6].(string)
		}
		invitations = append(invitations, Invitation{
			Name:     row[1].(string),
			Domicile: row[2].(string),
			Priority: priority,
			Invitee:  row[4].(string),
			Gender:   row[5].(string),
			Prefix:   prefix,
			Key:      sheetKey,
		})

	}

	if regenerateLink {
		regenerateInvivationLink(invitations)
	}

	invitationMap := make(InvitationMap)
	for _, inv := range invitations {
		invitationMap[inv.Key] = inv
	}

	dbClient, err := modules.FB.Database(ctx)
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

	detailRef := dbClient.NewRef("invitation/detail")
	err = detailRef.Set(ctx, &detail)
	if err != nil {
		log.Fatalln("Error updating database :", err)
		os.Exit(1)
	}
}
