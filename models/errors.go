package models

type AppError struct {
	Err     error
	Message string
	Status  int
}

func (e AppError) Error() string {
	return e.Message
}
