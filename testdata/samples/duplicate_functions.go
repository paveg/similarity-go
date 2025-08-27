package samples

import (
	"errors"
	"fmt"
	"strings"
)

// These functions are intentionally similar/duplicate for similarity detection testing

func ProcessUser(name string, age int) string {
	if name == "" {
		return "invalid name"
	}
	if age < 0 {
		return "invalid age"
	}
	return fmt.Sprintf("User: %s, Age: %d", name, age)
}

func ProcessAdmin(username string, years int) string {
	if username == "" {
		return "invalid name"
	}
	if years < 0 {
		return "invalid age"
	}
	return fmt.Sprintf("User: %s, Age: %d", username, years)
}

func HandleRequest(data string) error {
	if data == "" {
		return errors.New("empty data")
	}

	// Process data
	processed := strings.ToUpper(data)
	fmt.Println(processed)

	return nil
}

func HandleCommand(input string) error {
	if input == "" {
		return errors.New("empty data")
	}

	// Process input
	processed := strings.ToUpper(input)
	fmt.Println(processed)

	return nil
}
