package sheets

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Writer struct {
	srv           *sheets.Service
	spreadsheetID string
	sheetName     string
}

func NewWriter(ctx context.Context, spreadsheetID, sheetName string) (*Writer, error) {
	credsJSON := os.Getenv("GOOGLE_CREDENTIALS")
	if credsJSON == "" {
		return nil, fmt.Errorf("GOOGLE_CREDENTIALS not set")
	}

	creds, err := google.CredentialsFromJSON(ctx, []byte(credsJSON), sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("credentials error: %w", err)
	}

	srv, err := sheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("sheets.NewService error: %w", err)
	}

	return &Writer{
		srv:           srv,
		spreadsheetID: spreadsheetID,
		sheetName:     sheetName,
	}, nil
}

func (w *Writer) ClearSheet(ctx context.Context) error {
	clearRange := w.sheetName
	_, err := w.srv.Spreadsheets.Values.Clear(w.spreadsheetID, clearRange, &sheets.ClearValuesRequest{}).Context(ctx).Do()
	return err
}

func (w *Writer) WriteData(ctx context.Context, values [][]interface{}) error {
	writeRange := w.sheetName + "!A1"

	_, err := w.srv.Spreadsheets.Values.Append(
		w.spreadsheetID,
		writeRange,
		&sheets.ValueRange{
			Values: values,
		},
	).ValueInputOption("RAW").Do()
	if err != nil {
		return fmt.Errorf("failed to append values: %w", err)
	}

	return nil
}
