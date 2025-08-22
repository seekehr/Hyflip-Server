package main

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/config"
	"Hyflip-Server/internal/env"
	"Hyflip-Server/internal/flippers"
	"Hyflip-Server/internal/routes"
	"Hyflip-Server/internal/storage"
	"bufio"
	"encoding/json"
	"github.com/labstack/echo/v4"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ResponseType struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// For testing purposes.
func main() {
	logFile := initLogs()
	defer logFile.Close()
	// Init env
	env.InitEnv()
	key := os.Getenv(env.INTERNAL_HYPIXEL_API_KEY)
	if key == "" {
		panic("Internal Hypixel API key not found in env.")
	}

	// Init API client
	cl := api.Init(key)
	checkKey(cl)

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

	// Start echo in a goroutine so we don't block our command loop ;3
	go func() {
		e.Logger.Fatal(e.Start(":3000"))
	}()

	token := os.Getenv("TOKEN")
	if token == "" {
		log.Println("Token not found in .env.")
		log.Println("Attempting to create account. Add the token in .env and reload...")
		time.Sleep(250 * time.Millisecond) // wait for server to start
		createAccount("Seekher")
		return
	}

	conf, err := loadConfigs(token, configTable)
	if err != nil {
		log.Println("Error loading config. Error: " + err.Error())
		return
	}

	commandLoop(cl, conf)
}

func checkKey(cl *api.HypixelApiClient) {
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

func initLogs() *os.File {
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

func loadConfigs(token string, configTable *storage.ConfigTableClient) (*config.UserConfig, error) {
	return configTable.GetConfig(token)
}

func commandLoop(cl *api.HypixelApiClient, config *config.UserConfig) {
	time.Sleep(230 * time.Millisecond) // for our > to LIKELY appear below the 'http server started at'
	reader := bufio.NewReader(os.Stdin)

	for {
		log.Print("> ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "cracc "):
			parts := strings.SplitN(line, " ", 2)
			if len(parts) < 2 {
				log.Println("Usage: create_account <username>")
				continue
			}

			username := parts[1]
			createAccount(username)
		case line == "bzflip":
			timeStart := time.Now()
			flips, err := flippers.Flip(cl, &config.BzConfig)
			if err != nil {
				panic(err)
			}
			for _, flip := range flips {
				log.Printf("\nFound flip! Id: %s, profit: %d.", flip.ItemId, flip.Profit)
			}
			log.Println("Flipping complete in ", time.Since(timeStart))
		case line == "exit":
			log.Println("Exiting...")
			return
		default:
			log.Println("Unknown command. Available: cracc <username>, bzflip, exit")
		}
	}
}

// createAccount - convenience sake.
func createAccount(username string) {
	var response ResponseType
	err := makeRequestAndReadResp(true,
		"http://localhost:3000/create_account",
		strings.NewReader("{\"username\":\""+username+"\"}"),
		&response)
	if err != nil {
		log.Println("Error creating account. Error: " + err.Error())
		return
	}

	if response.Data == nil {
		log.Printf("\nEmpty response data. Could not create account. Message: %s. Success: %t. \n", response.Message, response.Success)
		return
	}

	tokenMap := response.Data.(map[string]interface{})
	key, ok := tokenMap["key"].(string)
	if !ok {
		log.Println("User key not found or invalid type. Empty response.")
		return
	}

	hash := storage.GetHash(key, username)
	log.Printf("Account created for %s. Key: %s\n", username, hash)
	log.Println(hash)
}

func makeRequestAndReadResp(post bool, url string, body *strings.Reader, dst any) error {
	var (
		resp *http.Response
		err  error
	)

	// this looks ugly asl but ME DONT CARE HAHA
	if post {
		resp, err = http.Post(url, "application/json", body)
	} else {
		resp, err = http.Get(url) // GET cannot have a body
	}

	if err != nil {
		log.Println("Error:", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response:", err)
		return err
	}
	if err := json.Unmarshal(respBody, dst); err != nil {
		log.Println("Error unmarshalling response:", err)
		return err
	}

	return nil
}
