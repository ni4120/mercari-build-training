package app

import (
	"context"
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

var errImageNotFound = errors.New("image not found")

type Item struct {
	ID       int    `db:"id" json:"-"`
	Name     string `db:"name" json:"name"`
	Category string `db:"category" json:"category"`
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

// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
	_, err := i.db.ExecContext(ctx, "INSERT INTO items (name, category, image_name) VALUES (?, ?, ?)", item.Name, item.Category, item.Image)
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
	rows, err := i.db.QueryContext(ctx, "SELECT id, name, category, image_name FROM items")
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
	err := i.db.QueryRowContext(ctx, "SELECT id, name, category, image_name FROM items WHERE id = ?", itemId).
		Scan(&item.ID, &item.Name, &item.Category, &item.Image)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Item{}, errors.New("item not found")
		}
		return Item{}, err
	}
	return item, nil
}

func (i *itemRepository) SearchItemsByKeyword(ctx context.Context, keyword string) ([]Item, error) {
	rows, err := i.db.QueryContext(ctx, "SELECT id,name,category,image_name FROM items WHERE name LIKE ?", "%"+keyword+"%")
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
