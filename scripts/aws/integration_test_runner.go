//go:build ignore

// integration_test_runner.go exercises the DynamoDB UserReader against a real
// AWS DynamoDB table.  Run via scripts/aws/integration-test.sh.
//
// Required env vars:
//   AWS_REGION    – AWS region (default: us-east-1)
//   TABLE_NAME    – DynamoDB table name (default: users)
//   DOCUMENT_INDEX - GSI name for document lookups (default: document-index)
//   TEST_DOCUMENT  – document of the seeded test user
//   TEST_PASSWORD  – plaintext password to validate

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbAdapter "github.com/soat13/ms-auth/internal/auth/infra/out/dynamodb"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	region := getEnv("AWS_REGION", "us-east-1")
	tableName := getEnv("TABLE_NAME", "users")
	documentIndex := getEnv("DOCUMENT_INDEX", "document-index")
	testDocument := getEnv("TEST_DOCUMENT", "12345678909")
	testPassword := getEnv("TEST_PASSWORD", "TestPass123!")

	ctx := context.Background()

	// Load default AWS credential chain (env vars, ~/.aws, instance role, etc.)
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		fatalf("load AWS config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)
	repo := ddbAdapter.NewUserRepository(client, tableName, documentIndex)

	fmt.Printf("==> GetByDocument(%q) ...\n", testDocument)

	user, err := repo.GetByDocument(ctx, testDocument)
	if err != nil {
		fatalf("GetByDocument: %v", err)
	}
	if user == nil {
		fatalf("user not found — did you run seed-test-user.sh?")
	}

	fmt.Printf("  ID       : %s\n", user.ID)
	fmt.Printf("  Name     : %s\n", user.Name)
	fmt.Printf("  Email    : %s\n", user.Email)
	fmt.Printf("  Document : %s\n", testDocument)
	fmt.Printf("  Roles    : %v\n", user.Roles.Strings())

	fmt.Printf("\n==> Validating password ...\n")
	if !user.Password.Matches(testPassword) {
		fatalf("password validation FAILED — check the seeded hash")
	}

	fmt.Println("\n✅  Integration test passed!")
	fmt.Println("   The DynamoDB adapter is reading and mapping users correctly.")
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "❌  FAIL: "+format+"\n", args...)
	os.Exit(1)
}
