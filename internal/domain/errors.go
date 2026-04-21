package domain

import "fmt"

type BusinessError struct {
	Code        string
	Description string
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Description)
}

var ErrDateMustBeToday = &BusinessError{
	Code:        "GMS-001",
	Description: "date must be today",
}

var ErrEntryAlreadyProcessing = &BusinessError{
	Code:        "GMS-002",
	Description: "entry is not in created status and cannot be processed",
}

var ErrEntryAlreadyPublished = &BusinessError{
	Code:        "GMS-003",
	Description: "entry already published",
}
