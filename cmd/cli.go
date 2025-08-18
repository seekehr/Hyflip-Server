package main

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/env"
	"Hyflip-Server/internal/routes"
	"Hyflip-Server/internal/storage"
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
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
	userDb := storage.InitUserDb()
	defer userDb.Close()
	fmt.Println("Connected to DB.")
	configTable := storage.InitConfigTable(userDb)
	defer configTable.Close()
	fmt.Println("Initialized config table.")

	// Register routes
	e := echo.New()
	e.HideBanner = true
	routes.RegisterRoutes(e, userDb, cl)
	fmt.Println("Registered routes.")

	// Start echo in a goroutine so we don't block our command loop ;3
	go func() {
		e.Logger.Fatal(e.Start(":3000"))
	}()

	commandLoop()
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

func commandLoop() {
	time.Sleep(500 * time.Millisecond) // for our > to LIKELY appear below the 'http server started at'
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
			var response ResponseType
			err := makeRequestAndReadResp(true,
				"http://localhost:3000/create_account",
				strings.NewReader("{\"username\":\""+username+"\"}"),
				&response)
			if err != nil {
				continue
			}

			if response.Data == nil {
				fmt.Printf("\nEmpty response data. Could not create account. Message: %s. Success: %t. \n", response.Message, response.Success)
				continue
			}

			token := response.Data.(map[string]interface{})
			if _, ok := token["key"]; !ok {
				fmt.Println("User key not found. Empty response.")
				continue
			}

			fmt.Printf("Account created for %s. Key: %s\n", username, token)
			fmt.Println(token)
		case line == "exit":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Unknown command. Available: cracc <username>, exit")
		}
	}
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
