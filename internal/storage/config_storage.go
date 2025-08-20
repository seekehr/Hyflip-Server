package storage

import (
	"Hyflip-Server/internal/config"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
)

const CreateUserConfigTableQuery = `
CREATE TABLE IF NOT EXISTS user_configs (
    user_key_hash CHAR(64) PRIMARY KEY,
    ahconfig JSONB NOT NULL,
    bzconfig JSONB NOT NULL
);
`

const UpsertUserConfigQuery = `
INSERT INTO user_configs (user_key_hash, ahconfig, bzconfig)
VALUES ($1, $2, $3)
ON CONFLICT (user_key_hash) DO UPDATE
SET ahconfig = EXCLUDED.ahconfig,
    bzconfig = EXCLUDED.bzconfig;
`

const GetUserConfigQuery = `
SELECT ahconfig, bzconfig FROM user_configs WHERE user_key_hash = $1;
`

type ConfigTableClient struct {
	pool *pgxpool.Pool
}

// InitConfigTable initializes ConfigTableClient
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

func (cl *ConfigTableClient) SaveConfig(userKeyHash, userConfig *config.UserConfig) error {
	// converting to jsonb to save
	ahConfig, err := json.Marshal(userConfig.AhConfig)
	if err != nil {
		return err
	}
	bzConfig, err := json.Marshal(userConfig.BzConfig)
	if err != nil {
		return err
	}

	ctx, cancel := getContext()
	defer cancel()
	_, err = cl.pool.Exec(ctx, UpsertUserConfigQuery, userKeyHash, ahConfig, bzConfig)
	return err
}

func (cl *ConfigTableClient) GetConfig(userKeyHash string) (*config.UserConfig, error) {
	ctx, cancel := getContext()
	defer cancel()

	var cfg config.UserConfig
	err := cl.pool.QueryRow(ctx, GetUserConfigQuery, userKeyHash).Scan(&cfg.AhConfig, &cfg.BzConfig)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (cl *ConfigTableClient) Close() {
	cl.pool.Close()
}
