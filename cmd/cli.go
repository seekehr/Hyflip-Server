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
	"fmt"
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
	initLogs()
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
	fmt.Println("Connected to DB.")
	configTable := storage.InitConfigTable(userDb)
	defer configTable.Close()
	fmt.Println("Initialized config table.")

	// Register routes
	e := echo.New()
	e.HideBanner = true
	routes.RegisterRoutes(e, userDb, cl, configTable)
	fmt.Println("Registered routes.")

	// Start echo in a goroutine so we don't block our command loop ;3
	go func() {
		e.Logger.Fatal(e.Start(":3000"))
	}()

	token := os.Getenv("TOKEN")
	if token == "" {
		fmt.Println("Token not found in .env.")
		createAccount("Seekher")
		fmt.Println("Attempting to create account. Add the token in .env and reload...")
		return
	}

	conf, err := loadConfigs(token, configTable)
	if err != nil {
		fmt.Println("Error loading config. Error: " + err.Error())
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
		fmt.Println("API Key is valid. Proceeding...")
	} else {
		panic("api key is invalid")
	}
}

func initLogs() {
	os.Mkdir("logs", os.ModePerm)
	file, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Redirect all log output to the file
	log.SetOutput(file)
}

func loadConfigs(token string, configTable *storage.ConfigTableClient) (*config.UserConfig, error) {
	return configTable.GetConfig(token)
}

func commandLoop(cl *api.HypixelApiClient, config *config.UserConfig) {
	time.Sleep(230 * time.Millisecond) // for our > to LIKELY appear below the 'http server started at'
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "cracc "):
			parts := strings.SplitN(line, " ", 2)
			if len(parts) < 2 {
				fmt.Println("Usage: create_account <username>")
				continue
			}

			username := parts[1]
			createAccount(username)
		case line == "bzflip":
			flips, err := flippers.Flip(cl, &config.BzConfig)
			if err != nil {
				panic(err)
			}
			for _, flip := range flips {
				fmt.Printf("\nFound flip! Id: %s, profit: %d.", flip.ItemId, flip.Profit)
			}
		case line == "exit":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Unknown command. Available: cracc <username>, bzflip, exit")
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
		fmt.Println("Error creating account. Error: " + err.Error())
		return
	}

	if response.Data == nil {
		fmt.Printf("\nEmpty response data. Could not create account. Message: %s. Success: %t. \n", response.Message, response.Success)
		return
	}

	tokenMap := response.Data.(map[string]interface{})
	key, ok := tokenMap["key"].(string)
	if !ok {
		fmt.Println("User key not found or invalid type. Empty response.")
		return
	}

	hash := storage.GetHash(key, username)
	fmt.Printf("Account created for %s. Key: %s\n", username, hash)
	fmt.Println(hash)
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
		fmt.Println("Error:", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return err
	}
	if err := json.Unmarshal(respBody, dst); err != nil {
		fmt.Println("Error unmarshalling response:", err)
		return err
	}

	return nil
}
