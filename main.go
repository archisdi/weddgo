package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"weddgo/modules"

	"github.com/gosimple/slug"
	"github.com/joho/godotenv"
	"google.golang.org/api/sheets/v4"
)

var SheetID string

func regenerateInvivationLink(index int, key string, wg *sync.WaitGroup) {
	col := strconv.Itoa(index)
	var vr sheets.ValueRange

	link := os.Getenv("INVITATION_BASE_URL") + "/" + key
	myval := []interface{}{link}
	vr.Values = append(vr.Values, myval)
	modules.SH.Spreadsheets.Values.Update(SheetID, os.Getenv("LINK_SHEET_COL")+col, &vr).ValueInputOption("RAW").Do()

	fmt.Println(link)
}

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

	var wg sync.WaitGroup
	for i, row := range sheet.Values {
		priority, _ := strconv.Atoi(row[2].(string))

		name := row[0].(string)
		sheetIndex := i + 2
		sheetKey := slug.Make(name)

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
			Key:      sheetKey,
		})

		if regenerateLink {
			wg.Add(1)
			regenerateInvivationLink(sheetIndex, sheetKey, &wg)
		}
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

	detailRef := dbClient.NewRef("invitation/digital")
	err = detailRef.Set(ctx, &detail)
	if err != nil {
		log.Fatalln("Error updating database :", err)
		os.Exit(1)
	}
}
