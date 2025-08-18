package env

import "github.com/joho/godotenv"

// INTERNAL_HYPIXEL_API_KEY - anything for the autocomplete huh
const INTERNAL_HYPIXEL_API_KEY = "INTERNAL_HYPIXEL_API_KEY"

// InitEnv - Load the .env... what else?
func InitEnv() {
	env := godotenv.Load()
	if env != nil {
		panic(".env file not found")
	}
}
