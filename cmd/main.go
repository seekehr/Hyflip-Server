package main

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/env"
	"Hyflip-Server/internal/routes"
	"Hyflip-Server/internal/storage"
	"github.com/labstack/echo/v4"
	"io"
	"log"
	"os"
)

// For testing purposes.
func main() {
	logFile := initialiseLogs()
	defer logFile.Close()
	// Init env
	env.InitEnv()
	key := os.Getenv(env.INTERNAL_HYPIXEL_API_KEY)
	if key == "" {
		panic("Internal Hypixel API key not found in env.")
	}

	// Init API client
	cl := api.Init(key)
	verifyKey(cl)

	// Init DB
	userDb := storage.InitDb()
	defer userDb.Close()
	log.Println("Connected to DB.")
	configTable := storage.InitConfigTable(userDb)
	defer configTable.Close()
	log.Println("Initialized config table.")

	// Register routes
	e := echo.New()
	e.HideBanner = true
	routes.RegisterRoutes(e, userDb, cl, configTable)
	log.Println("Registered routes.")

	e.Logger.Fatal(e.Start(":3000"))
}

func verifyKey(cl *api.HypixelApiClient) {
	valid, err := api.CheckApiKey(cl)
	if err != nil {
		panic(err)
	}

	if valid {
		log.Println("API Key is valid. Proceeding...")
	} else {
		panic("api key is invalid")
	}
}

func initialiseLogs() *os.File {
	os.Mkdir("logs", os.ModePerm)
	file, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	// writes to both file and stdout
	multi := io.MultiWriter(file, os.Stdout)

	log.SetOutput(multi)
	return file
}
