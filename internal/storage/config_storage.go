package storage

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

const CreateUserConfigTableQuery = `
CREATE TABLE IF NOT EXISTS user_configs (
    user_key_hash CHAR(64) PRIMARY KEY,
    config JSONB NOT NULL
);
`

const UpsertUserConfigQuery = `
INSERT INTO user_configs (user_key_hash, config)
VALUES ($1, $2)
ON CONFLICT (user_key_hash) DO UPDATE
SET config = EXCLUDED.config;
`

const GetUserConfigQuery = `
SELECT config FROM user_configs WHERE user_key_hash = $1;
`

type ConfigTableClient struct {
	pool *pgxpool.Pool
}

// InitConfigTable is used to initialize the ConfigTableClient. It uses UserDatabaseClient as both use the same `Hyflip` database and so a separate connection pool is not needed
func InitConfigTable(cl *UserDatabaseClient) *ConfigTableClient {
	ctx, cancel := getContext()
	defer cancel()

	_, err := cl.pool.Exec(ctx, CreateUserConfigTableQuery)
	if err != nil {
		panic("Unable to create user_configs table: " + err.Error())
	}

	return &ConfigTableClient{
		pool: cl.pool,
	}
}

// SaveConfig stores or updates a user's config
func (cl *ConfigTableClient) SaveConfig(userKeyHash string, configJSON string) error {
	ctx, cancel := getContext()
	defer cancel()

	_, err := cl.pool.Exec(ctx, UpsertUserConfigQuery, userKeyHash, configJSON)
	return err
}

func (cl *ConfigTableClient) GetConfig(userKeyHash string) (string, error) {
	ctx, cancel := getContext()
	defer cancel()

	var configJSON string
	err := cl.pool.QueryRow(ctx, GetUserConfigQuery, userKeyHash).Scan(&configJSON)
	if err != nil {
		return "", err
	}
	return configJSON, nil
}

func (cl *ConfigTableClient) Close() {
	cl.pool.Close()
}
