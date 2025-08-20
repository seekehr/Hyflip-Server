package storage

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strconv"
	"time"
)

// in case of a db leak, we're using the hash here to prevent damage

const CreateUsersTableQuery = `
CREATE TABLE IF NOT EXISTS users (
    user_key_hash CHAR(64) PRIMARY KEY,
    user_uuid TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL
);
`

const InsertUserQuery = `
INSERT INTO users (user_key_hash, user_uuid, username)
VALUES ($1, $2, $3)
ON CONFLICT (user_key_hash) DO UPDATE
SET user_uuid = EXCLUDED.user_uuid,
    username = EXCLUDED.username;
`

const GetUserQuery = `
SELECT user_uuid, username FROM users WHERE user_key_hash = $1;
`

const GetKeyFromUserQuery = `
SELECT user_key_hash FROM users WHERE username = $1;
`

const DeleteUserExistingKeyQuery = `
    DELETE FROM users WHERE username = $1;
`
const UserExistsQuery = `
SELECT true FROM tokens WHERE token = $1 LIMIT 1;
`

type DatabaseClient struct {
	pool *pgxpool.Pool
}

func InitDb() *DatabaseClient {
	ctx, cancel := getContext()
	defer cancel()
	dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))

	if err != nil {
		panic("Unable to create connection pool: " + err.Error())
	}

	otherCtx, otherCancel := getContext()
	defer otherCancel()
	_, err = dbpool.Exec(otherCtx, CreateUsersTableQuery)
	if err != nil {
		panic("unable to create users table: " + err.Error())
	}
	return &DatabaseClient{
		pool: dbpool,
	}
}

// DeleteAnyExistingUserKey Usually used upon registration to make sure one user doesn't have multiple keys.
func (cl *DatabaseClient) DeleteAnyExistingUserKey(username string) error {
	ctx, cancel := getContext()
	defer cancel()
	_, err := cl.pool.Exec(ctx, DeleteUserExistingKeyQuery, username)
	if err != nil {
		return err
	}
	return nil
}

// CreateUser Registers a new key for the user.
func (cl *DatabaseClient) CreateUser(userKeyHash string, uuid string, username string) error {
	ctx, cancel := getContext()
	defer cancel()
	err := cl.DeleteAnyExistingUserKey(username)
	if err != nil {
		return err
	}
	rowsAffected, err := cl.pool.Exec(ctx, InsertUserQuery, userKeyHash, uuid, username)
	if err != nil {
		return err
	}

	fmt.Println("Rows affected: " + strconv.Itoa(int(rowsAffected.RowsAffected())))
	return nil
}

// GetUser returns the user UUID and hash from their key.
func (cl *DatabaseClient) GetUser(userKeyHash string) (string, string, error) {
	var uuid, username string
	ctx, cancel := getContext()
	err := cl.pool.QueryRow(
		ctx,
		GetUserQuery,
		userKeyHash,
	).Scan(&uuid, &username)
	defer cancel()

	if err != nil {
		return "", "", err
	}

	return uuid, username, nil
}

// GetKeyFromUser returns the hashed key for a given username.
func (cl *DatabaseClient) GetKeyFromUser(username string) (string, error) {
	ctx, cancel := getContext()
	defer cancel()

	var keyHash string
	err := cl.pool.QueryRow(ctx, GetKeyFromUserQuery, username).Scan(&keyHash)
	if err != nil {
		return "", err
	}

	return keyHash, nil
}

func (cl *DatabaseClient) ExistsUser(userKeyHash string) (bool, error) {
	ctx, cancel := getContext()
	defer cancel()
	var exists bool
	err := cl.pool.QueryRow(ctx, UserExistsQuery, userKeyHash).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (cl *DatabaseClient) Close() {
	cl.pool.Close()
}

// getContext 5 second timeout to prevent hanging.
func getContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
