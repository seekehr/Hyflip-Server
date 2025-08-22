package storage

import (
	"Hyflip-Server/internal/config"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const CreateUserConfigTableQuery = `
CREATE TABLE IF NOT EXISTS user_configs (
    user_key_hash VARCHAR(44) PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    ahconfig JSONB NOT NULL,
    bzconfig JSONB NOT NULL
);
`

const GetUserConfigByUserKeyHashQuery = `
SELECT ahconfig, bzconfig FROM user_configs WHERE user_key_hash = $1;
`

const GetUserConfigByUsernameQuery = `
SELECT ahconfig, bzconfig FROM user_configs WHERE username = $1;
`

const DeleteUserConfigByUsernameQuery = `
DELETE FROM user_configs WHERE username = $1;
`

const InsertUserConfigQuery = `
INSERT INTO user_configs (user_key_hash, username, ahconfig, bzconfig)
VALUES ($1, $2, $3, $4);
`

type ConfigTableClient struct {
	pool *pgxpool.Pool
}

// InitConfigTable initializes ConfigTableClient
func InitConfigTable(cl *DatabaseClient) *ConfigTableClient {
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

// SaveConfig handles all cases to save a config. For example, use nil instead of userConfig if you want to save a default config.
// Handles new hashes for usernames correctly, and transfers the config to the new hash accordingly. Or to add a specific config (i.e user designed config).
func (cl *ConfigTableClient) SaveConfig(userKeyHash string, username string, userConfig *config.UserConfig) error {
	ctx, cancel := getContext()
	defer cancel()

	// A specific config is provided. This is a direct update
	if userConfig != nil {
		return cl.upsertConfig(ctx, userKeyHash, username, userConfig)
	}

	// If no config is provided, check if one exists for the username
	existingConfig, err := cl.GetConfigByUsername(username)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if err == nil {
		// No config provided, but one exists. Transfer the existing config
		return cl.transferConfig(ctx, userKeyHash, username, existingConfig)
	}

	// No config provided and none exists. Save a new default config
	return cl.saveDefaultConfig(ctx, userKeyHash, username)
}

// saveDefaultConfig creates and saves a new default configuration for a user.
func (cl *ConfigTableClient) saveDefaultConfig(ctx context.Context, userKeyHash string, username string) error {
	defaultConfig := &config.UserConfig{
		AhConfig: *config.GenerateDefaultAHConfig(),
		BzConfig: *config.GenerateDefaultBZConfig(),
	}
	return cl.upsertConfig(ctx, userKeyHash, username, defaultConfig)
}

// transferConfig effectively re-associates an existing configuration with a new userKeyHash.
func (cl *ConfigTableClient) transferConfig(ctx context.Context, newUserKeyHash string, username string, existingConfig *config.UserConfig) error {
	return cl.upsertConfig(ctx, newUserKeyHash, username, existingConfig)
}

// upsertConfig universal function that uses transactions to insert a config.
func (cl *ConfigTableClient) upsertConfig(ctx context.Context, userKeyHash string, username string, userConfig *config.UserConfig) error {
	ahConfigJSON, err := json.Marshal(userConfig.AhConfig)
	if err != nil {
		return err
	}
	bzConfigJSON, err := json.Marshal(userConfig.BzConfig)
	if err != nil {
		return err
	}

	tx, err := cl.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, DeleteUserConfigByUsernameQuery, username)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, InsertUserConfigQuery, userKeyHash, username, ahConfigJSON, bzConfigJSON)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetConfig retrieves a user's config by their user_key_hash (also used in users_table)
func (cl *ConfigTableClient) GetConfig(userKeyHash string) (*config.UserConfig, error) {
	ctx, cancel := getContext()
	defer cancel()

	var ahConfigRaw, bzConfigRaw []byte
	err := cl.pool.QueryRow(ctx, GetUserConfigByUserKeyHashQuery, userKeyHash).Scan(&ahConfigRaw, &bzConfigRaw)
	if err != nil {
		return nil, err
	}
	return unmarshalConfig(ahConfigRaw, bzConfigRaw)
}

// GetConfigByUsername retrieves a user's config by their username
func (cl *ConfigTableClient) GetConfigByUsername(username string) (*config.UserConfig, error) {
	ctx, cancel := getContext()
	defer cancel()

	var ahConfigRaw, bzConfigRaw []byte
	err := cl.pool.QueryRow(ctx, GetUserConfigByUsernameQuery, username).Scan(&ahConfigRaw, &bzConfigRaw)
	if err != nil {
		return nil, err
	}
	return unmarshalConfig(ahConfigRaw, bzConfigRaw)
}

// unmarshalConfig to reduce code duplication
func unmarshalConfig(ahConfigRaw, bzConfigRaw []byte) (*config.UserConfig, error) {
	var cfg config.UserConfig
	if err := json.Unmarshal(ahConfigRaw, &cfg.AhConfig); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bzConfigRaw, &cfg.BzConfig); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (cl *ConfigTableClient) Close() {
	cl.pool.Close()
}
