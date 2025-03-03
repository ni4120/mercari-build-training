package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	// STEP 5-1: uncomment this line
	// _ "github.com/mattn/go-sqlite3"
)

var errImageNotFound = errors.New("image not found")

type Item struct {
	ID       int    `db:"id" json:"-"`
	Name     string `db:"name" json:"name"`
	Category string `db:"category" json:"category"`
	Image    string `db:"image" json:"image"`
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
	GetAllItem(ctx context.Context) ([]Item, error)
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	// fileName is the path to the JSON file storing items.
	fileName string
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository() ItemRepository {
	return &itemRepository{fileName: "items.json"}
}

// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
	// STEP 4-2: add an implementation to store an item
	items, err := i.GetAllItem(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			items = []Item{}
		} else {
			return err
		}
	}

	items = append(items, *item)

	file, err := os.Create(i.fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")
	if err := encoder.Encode(items); err != nil {
		return err
	}

	return nil
}

// StoreImage stores an image and returns an error if any.
// This package doesn't have a related interface for simplicity.
func StoreImage(fileName string, image []byte) error {
	// STEP 4-4: add an implementation to store an image
	hash := sha256.Sum256([]byte(fileName))
	fileName = hex.EncodeToString(hash[:]) + ".jpg"

	filePath := filepath.Join("images", fileName)

	if err := os.WriteFile(filePath, image, 0644); err != nil {
		return err
	}

	return nil
}

func (i *itemRepository) GetAllItem(ctx context.Context) ([]Item, error) {
	file, err := os.Open(i.fileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Item{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var items []Item

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&items); err != nil {
		if errors.Is(err, io.EOF) {
			return []Item{}, nil
		}
		return nil, err
	}
	return items, nil

}
