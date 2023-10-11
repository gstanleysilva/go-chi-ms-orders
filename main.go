package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/guhstanley/go-chi-ms-orders/application"
)

func main() {
	app := application.New()

	//Creates a base context for the app based on a root level and a signal that will be
	//notified if an interruption occur
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	//Cancell all chield level contexts at the end of the function
	defer cancel()

	err := app.Start(ctx)

	if err != nil {
		fmt.Println("failed to start app:", err)
	}
}
