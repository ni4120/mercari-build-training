package app

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/mock/gomock"
)

const (
	testImageData = "test.jpg"
)

func TestParseAddItemRequest(t *testing.T) {
	t.Parallel()

	type wants struct {
		req *AddItemRequest
		err bool
	}

	// STEP 6-1: define test cases
	cases := map[string]struct {
		args      map[string]string
		imageData []byte
		wants
	}{
		"ok: valid request": {
			args: map[string]string{
				"name":     "Test Item",
				"category": "Test Category",
			},
			imageData: []byte(testImageData),
			wants: wants{
				req: &AddItemRequest{
					Name:     "Test Item",
					Category: "Test Category",
					Image:    []byte(testImageData),
				},
				err: false,
			},
		},
		"ng: empty request": {
			args:      map[string]string{},
			imageData: nil,
			wants: wants{
				req: nil,
				err: true,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// prepare request body
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			for k, v := range tt.args {
				if err := writer.WriteField(k, v); err != nil {
					t.Fatalf("failed to write field %s: %v", k, err)
				}

			}

			// Add image data
			if len(tt.imageData) > 0 {
				part, err := writer.CreateFormFile("image", "testdata/test.jpg")
				if err != nil {
					t.Fatalf("failed to create file part: %v", err)
				}
				if _, err := part.Write(tt.imageData); err != nil {
					t.Fatalf("failed to write image data: %v", err)
				}
			}

			writer.Close()

			// prepare HTTP request
			req, err := http.NewRequest("POST", "http://localhost:9000/items", body)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// execute test target
			got, err := parseAddItemRequest(req)

			// confirm the result
			if err != nil {
				if !tt.err {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if diff := cmp.Diff(tt.wants.req, got); diff != "" {
				t.Errorf("unexpected request (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelloHandler(t *testing.T) {
	t.Parallel()

	// Please comment out for STEP 6-2
	// predefine what we want
	type wants struct {
		code int               // desired HTTP status code
		body map[string]string // desired body
	}
	want := wants{
		code: http.StatusOK,
		body: map[string]string{"message": "Hello, world!"},
	}

	// set up test
	req := httptest.NewRequest("GET", "/hello", nil)
	res := httptest.NewRecorder()

	h := &Handlers{}
	h.Hello(res, req)

	// STEP 6-2: confirm the status code
	if res.Code != want.code {
		t.Errorf("unexpected status code: got %d, want %d", res.Code, want.code)
	}
	// STEP 6-2: confirm response body
	var got map[string]string
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if diff := cmp.Diff(want.body, got); diff != "" {
		t.Errorf("unexpected response body (-want +got):\n%s", diff)
	}

}

func TestAddItem(t *testing.T) {
	t.Parallel()

	type wants struct {
		code int
	}
	cases := map[string]struct {
		args      map[string]string
		imageData []byte
		injector  func(m *MockItemRepository)
		wants
	}{
		"ok: correctly inserted": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
			},
			imageData: []byte(testImageData),
			injector: func(m *MockItemRepository) {
				m.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					Return(nil).Times(1)
			},
			wants: wants{
				code: http.StatusOK,
			},
		},
		"ng: failed to insert": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
			},
			imageData: nil,
			injector: func(m *MockItemRepository) {
				m.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			wants: wants{
				code: http.StatusInternalServerError,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockIR := NewMockItemRepository(ctrl)
			tt.injector(mockIR)

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			for k, v := range tt.args {
				if err := writer.WriteField(k, v); err != nil {
					t.Fatalf("failed to write field: %v", err)
				}
			}

			part, err := writer.CreateFormFile("image", "test.jpg")
			if err != nil {
				t.Fatalf("failed to create file part: %v", err)
			}

			if _, err := part.Write([]byte(testImageData)); err != nil {
				t.Fatalf("failed to write image data: %v", err)
			}

			writer.Close()

			req := httptest.NewRequest("POST", "/items", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rr := httptest.NewRecorder()
			h := &Handlers{itemRepo: mockIR}
			h.AddItem(rr, req)

			if tt.wants.code != rr.Code {
				t.Errorf("expected status code %d, got %d", tt.wants.code, rr.Code)
			}
			if tt.wants.code >= 400 {
				return
			}

			for _, v := range tt.args {
				if !strings.Contains(rr.Body.String(), v) {
					t.Errorf("response body does not contain %s, got: %s", v, rr.Body.String())
				}
			}
		})
	}
}

// STEP 6-4: uncomment this test
func TestAddItemE2e(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	db, closers, err := setupDB(t)
	if err != nil {
		t.Fatalf("failed to set up database: %v", err)
	}
	t.Cleanup(func() {
		for _, c := range closers {
			c()
		}
	})

	type wants struct {
		code      int
		name      string
		category  string
		imageData []byte
	}
	cases := map[string]struct {
		args      map[string]string
		imageData []byte
		wants
	}{
		"ok: correctly inserted": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
			},
			imageData: []byte(testImageData),
			wants: wants{
				code: http.StatusOK,
			},
		},
		"ng: failed to insert due to empty name and image": {
			args: map[string]string{
				"name":     "",
				"category": "phone",
			},
			imageData: nil,
			wants: wants{
				code: http.StatusBadRequest,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			h := &Handlers{itemRepo: &itemRepository{db: db}}

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			for k, v := range tt.args {
				if err := writer.WriteField(k, v); err != nil {
					t.Errorf("failed to write field %s: %v", k, err)
				}
			}

			part, err := writer.CreateFormFile("image", "test.jpg")
			if err != nil {
				t.Errorf("failed to create file part: %v", err)
			}
			if _, err := part.Write([]byte(testImageData)); err != nil {
				t.Errorf("failed to write image data: %v", err)
			}

			writer.Close()

			req := httptest.NewRequest("POST", "/items", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rr := httptest.NewRecorder()
			h.AddItem(rr, req)

			if tt.wants.code != rr.Code {
				t.Errorf("expected status code %d, got %d", tt.wants.code, rr.Code)
			}
			if tt.wants.code >= 400 {
				return
			}
			for _, v := range tt.args {
				if !strings.Contains(rr.Body.String(), v) {
					t.Errorf("response body does not contain %s, got: %s", v, rr.Body.String())
				}
			}
			// STEP 6-4: check inserted data
			if rr.Code == http.StatusOK {
				var item AddItemRequest
				if err := json.NewDecoder(rr.Body).Decode(&item); err != nil {
					t.Errorf("failed to decode response body: %v", err)
				}
				if item.Name != tt.wants.name || item.Category != tt.wants.category || !bytes.Equal(item.Image, tt.wants.imageData) {
					t.Errorf("expected (name, category,imageData) = (%s, %s,%v), but got (%s, %s%v)", tt.wants.name, tt.wants.category, tt.wants.imageData, item.Name, item.Category, item.Image)
				}
			}
		})
	}
}

func setupDB(t *testing.T) (db *sql.DB, closers []func(), e error) {
	t.Helper()

	defer func() {
		if e != nil {
			for _, c := range closers {
				c()
			}
		}
	}()

	f, err := os.CreateTemp(".", "*.sqlite3")
	if err != nil {
		return nil, nil, err
	}
	closers = append(closers, func() {
		f.Close()
		os.Remove(f.Name())
	})

	db, err = sql.Open("sqlite3", f.Name())
	if err != nil {
		return nil, nil, err
	}
	closers = append(closers, func() {
		db.Close()
	})

	sqlFile, err := os.ReadFile("../db/items.sql")
	if err != nil {
		t.Errorf("failed to read SQL file: %v", err)
		return nil, nil, err
	}

	_, err = db.Exec(string(sqlFile))
	if err != nil {
		t.Errorf("failed to execute SQL schema: %v", err)
		return nil, nil, err
	}

	return db, closers, nil
}
