package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"google.golang.org/api/sheets/v4"
)

type Invitation struct {
	Name     string `json:"name"`
	Domicile string `json:"domicile"`
	Priority int    `json:"priority"`
	Invitee  string `json:"invitee"`
}

func main() {
	if envErr := godotenv.Load(); envErr != nil {
		log.Fatal("error while loading environment file")
		os.Exit(1)
	}

	sheetID := "19RQe_T-MsxBSOfnwlXRcRKvRRxq6kGu6OA6mmb20aEM"

	ctx := context.Background()
	sheetsService, err := sheets.NewService(ctx)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	readRange := "public!B2:E"
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

	for _, inv := range invitations {
		fmt.Println(inv)
	}

}
