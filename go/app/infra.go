package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	// STEP 5-1: uncomment this line
	// _ "github.com/mattn/go-sqlite3"
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
		return err
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
	if err := os.WriteFile(fileName, image, 0644); err != nil {
		return err
	}

	return nil
}

func (i *itemRepository) GetAllItem(ctx context.Context) ([]Item, error) {
	file, err := os.Open(i.fileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			file, err := os.Create(i.fileName)
			if err != nil {
				return nil, err
			}
			defer file.Close()
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

func (i *itemRepository) GetItemById(ctx context.Context, itemId string) (Item, error) {
	items, err := i.GetAllItem(ctx)
	if err != nil {
		return Item{}, err
	}

	index, err := strconv.Atoi(itemId)
	if err != nil {
		return Item{}, errors.New("invalid item ID format")
	}

	index = index - 1
	if index < 0 || index >= len(items) {
		return Item{}, errors.New("item index out of range")
	}

	return items[index], nil
}
