package bank

import "os"

type tokenerAccesser interface {
	accessToken() string
}

type bluevineAccesser struct{}

func (bluevineAccesser) accessToken() string {
	return os.Getenv("BLUEVINE_ACCESS_TOKEN")
}

type sandboxAccesser struct{}

func (sandboxAccesser) accessToken() string {
	return os.Getenv("SANDBOX_ACCESS_TOKEN")
}
