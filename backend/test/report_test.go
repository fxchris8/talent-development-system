package test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"backend/internal/config"

	"github.com/stretchr/testify/assert"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func SetupServer() *gin.Engine {
	viperConfig := viper.New()

	// viperConfig.SetConfigFile(".env")
	// viperConfig.AddConfigPath("./")
	// viperConfig.AddConfigPath("./..")
	wd, _ := os.Getwd()
	projectRoot := filepath.Join(wd, "..") // move up one dir from /test
	viperConfig.SetConfigFile(filepath.Join(projectRoot, ".env"))

	err := viperConfig.ReadInConfig()
	if err != nil {
		panic(err)
	}

	log := config.NewLogger(viperConfig)
	validate := config.NewValidator()
	app := config.NewGin(viperConfig, log)
	db := config.NewDatabase(viperConfig, log)

	// Bootstrap all configs
	config.Bootstrap(&config.BootstrapConfig{
		DB:       db,
		App:      app,
		Log:      log,
		Validate: validate,
		Config:   viperConfig,
	})

	return app
}

func TestExcel(t *testing.T) {
	// Setup File
	filePath := "data/test-data.xlsx"
	file, err := os.Open(filePath)
	if err != nil {
		// The sample workbook is not kept in the repository; without it there is nothing to test.
		t.Skipf("skipping: %s not available (%v)", filePath, err)
	}
	defer file.Close()

	// Build Multipart Form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test-data.xlsx")
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		panic(err)
	}
	writer.Close()

	app := SetupServer()

	request := httptest.NewRequest(http.MethodPost, "/reports/upload", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	app.ServeHTTP(recorder, request)
	response := recorder.Result()

	responseBody, _ := io.ReadAll(response.Body)

	fmt.Println(string(responseBody))
	result := string(responseBody)
	assert.Equal(t, "{\"code\":201,\"status\":\"Created\",\"data\":\"Reports Created Successfully\"}", result)
}
