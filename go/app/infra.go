package app

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var errImageNotFound = errors.New("image not found")

type Item struct {
	ID       int    `db:"id" json:"-"`
	Name     string `db:"name" json:"name"`
	Category string `db:"category_name" json:"category_name"`
	Image    string `db:"image_name" json:"image_name"`
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
	GetAllItem(ctx context.Context) ([]Item, error)
	GetItemById(ctx context.Context, itemId string) (Item, error)
	SearchItemsByKeyword(ctx context.Context, keyword string) ([]Item, error)
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	db *sql.DB
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository(db *sql.DB) ItemRepository {
	return &itemRepository{db: db}
}

func initDB(db *sql.DB) error {
	const sqlFile = "db/items.sql"

	if _, err := os.Stat(sqlFile); os.IsNotExist(err) {
		slog.Error("SQL file not found", "file", sqlFile)
		return errors.New("SQL file not found")
	}

	sqlBytes, err := os.ReadFile(sqlFile)
	if err != nil {
		slog.Error("Failed to read SQL file", "file", sqlFile, "error", err)
		return err
	}

	_, err = db.Exec(string(sqlBytes))
	if err != nil {
		slog.Error("Failed to execute SQL file", "error", err)
		return err
	}

	slog.Info("Database initialized successfully")
	return nil
}

// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {

	var categoryID int

	err := i.db.QueryRowContext(ctx, "SELECT id FROM categories WHERE name = ?", item.Category).Scan(&categoryID)
	if err != nil {
		res, err := i.db.ExecContext(ctx, "INSERT INTO categories (name) VALUES (?)", item.Category)
		if err != nil {
			return err
		}
		lastID, _ := res.LastInsertId()
		categoryID = int(lastID)
	}

	_, err = i.db.ExecContext(ctx, "INSERT INTO items (name,category_id, image_name) VALUES (?, ?, ?)", item.Name, categoryID, item.Image)
	return err
}

// StoreImage stores an image and returns an error if any.
// This package doesn't have a related interface for simplicity.
func StoreImage(fileName string, image []byte) error {
	// STEP 4-4: add an implementation to store an image
	if err := os.WriteFile(fileName, image, 0644); err != nil {
		return err
	}

	return nil
}

func (i *itemRepository) GetAllItem(ctx context.Context) ([]Item, error) {
	rows, err := i.db.QueryContext(ctx, `
		SELECT items.id, items.name, categories.name AS category_name ,items.image_name
		FROM items
		JOIN categories ON items.category_id = categories.id
		`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Image); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (i *itemRepository) GetItemById(ctx context.Context, itemId string) (Item, error) {
	var item Item
	err := i.db.QueryRowContext(ctx, `
	SELECT items.id,items.name,categories.name AS category_name,items.image_name
	FROM items
	JOIN categories ON items.category_id = categories.id
	WHERE items.id = ?
	`, itemId).Scan(&item.ID, &item.Name, &item.Category, &item.Image)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Item{}, errors.New("item not found")
		}
		return Item{}, err
	}
	return item, nil
}

func (i *itemRepository) SearchItemsByKeyword(ctx context.Context, keyword string) ([]Item, error) {
	rows, err := i.db.QueryContext(ctx, `
		SELECT items.id, items.name, categories.name AS category_name, items.image_name
		FROM items
		JOIN categories ON items.category_id = categories.id
		WHERE items.name LIKE ?`, "%"+keyword+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Image); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
