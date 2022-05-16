package modules

import (
	"context"
	"log"
	"os"

	"google.golang.org/api/sheets/v4"
)

var SH *sheets.Service

func InitializeSheet(ctx context.Context) {
	sheetsService, err := sheets.NewService(ctx)
	if err != nil {
		log.Fatalln("Error initializing sheet service:", err)
		os.Exit(1)
	}

	SH = sheetsService
}
