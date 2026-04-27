package gcputil

import (
	"context"
	"fmt"
	"log"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func CreateSpredsheetService(credentialsFilename string) (*sheets.Service, context.Context, error) {
	b, err := os.ReadFile(credentialsFilename)
	if err != nil {
		return nil, nil, err
	}

	sheetCtx := context.Background()

	// Define the necessary scopes (Read/Write)
	credentialsConfig, _ := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	client, _ := GetClientFromTokenOrWeb("secret/token.json", credentialsConfig)

	// Authenticate
	// srv, err := sheets.NewService(sheetCtx, option.WithCredentialsFile("../secret/service-account.json"))
	srv, err := sheets.NewService(sheetCtx, option.WithHTTPClient(client))
	return srv, sheetCtx, err
}

func TestSimpleAppendSpreadsheet(spreadsheetId string, sheetName string) {
	srv, _, err := CreateSpredsheetService("secret/credentials.json")
	if err != nil {
		log.Fatalf("Unable to create spreadsheet service: %v", err)
	}

	var vr sheets.ValueRange
	vr.Values = append(vr.Values, []interface{}{"1", "tes"})

	// Append the data to the sheet
	// USER_ENTERED means values are parsed as if typed by a human (dates/numbers converted)
	_, err = srv.Spreadsheets.Values.Append(spreadsheetId, sheetName, &vr).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Do()

	if err != nil {
		log.Fatalf("Unable to append data to sheet: %v", err)
	}

	fmt.Println("Successfully appended CSV data!")
}
