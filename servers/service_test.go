package servers

import (
	"errors"
	"testing"
)

func validInput() CreateInput {
	return CreateInput{
		Location:  "Amsterdam",
		Username:  "panel",
		Ip:        "203.0.113.10",
		ApiUrl:    "http://203.0.113.10:2053",
		Port:      443,
		InBoundID: 1,
	}
}

func TestValidateCreateInputAcceptsValidServer(t *testing.T) {
	if err := validateCreateInput(validInput()); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
}

func TestValidateCreateInputRejectsBadValues(t *testing.T) {
	cases := map[string]func(*CreateInput){
		"empty location": func(in *CreateInput) { in.Location = "  " },
		"empty ip":       func(in *CreateInput) { in.Ip = "" },
		"empty api url":  func(in *CreateInput) { in.ApiUrl = "" },
		"empty username": func(in *CreateInput) { in.Username = "" },
		"zero port":      func(in *CreateInput) { in.Port = 0 },
		"port too large": func(in *CreateInput) { in.Port = 70000 },
		"negative limit": func(in *CreateInput) { in.MaxClients = -1 },
		"negative port":  func(in *CreateInput) { in.Port = -1 },
	}

	for name, mutate := range cases {
		input := validInput()
		mutate(&input)

		err := validateCreateInput(input)
		if err == nil {
			t.Errorf("%s: expected an error, got nil", name)
			continue
		}

		// Ошибка должна быть распознаваемой, чтобы API вернул 400, а не 500.
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: error %v does not wrap ErrInvalidInput", name, err)
		}
	}
}
