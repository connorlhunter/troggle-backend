package main

import (
	"context"
	"encoding/json"
	"log"
	"github.com/aws/aws-lambda-go/lambda" 
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Request represents the JSON input
type Request struct {
	Email string `json:"email"` // User email to check
}

// Response represents the JSON output
type Response struct {
	StatusCode int    `json:"statusCode"`
	Exists     bool   `json:"exists"`
	Message    string `json:"message,omitempty"`
}

// UserExists checks if a user with the given email exists in the specified DynamoDB table.
// Returns true if the user exists, false otherwise.
func UserExists(email string, db *dynamodb.Client, tableName string) (bool, string) {
	existsMsg := "User exists"
	notExistsMsg := "User does not exist"

	log.Printf("Checking if user exists: %s in table %s", email, tableName)

	// use Global Secondary Index to lookup by email rather than cognito user_id
	indexName := "email-index"

	// Prepare DynamoDB Query input
	input := &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		IndexName:              aws.String(indexName),
		KeyConditionExpression: aws.String("email = :email"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":email": &types.AttributeValueMemberS{Value: email},
		},
	}

	// Fetch item from DynamoDB
	result, err := db.Query(context.TODO(), input)
	if err != nil {
		log.Printf("Error fetching item from DynamoDB: %v", err)
		return false, notExistsMsg
	}

	//  A Query returns a slice of items, so check its length
	if len(result.Items) > 0 {
		log.Printf("User found: %s", email)
		return true, existsMsg
	}

	log.Printf("User not found: %s", email)
	return false, notExistsMsg
}

// handler is the Lambda entry point. It receives an API Gateway event,
// extracts the email from the request body, checks DynamoDB, and returns JSON.
func handler(ctx context.Context, payload json.RawMessage) (Response, error) {
	var req Request

	// Parse JSON body from API Gateway request
	err := json.Unmarshal([]byte(payload), &req)
	if err != nil {
		return Response{
			StatusCode: 400,
			Exists:     false,
			Message:    "Invalid request",
		}, err

	}

	// Load AWS SDK config (credentials, region, etc.)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("Error loading AWS config: %v", err)
		return Response{
			StatusCode: 500,
			Exists:     false,
			Message:    "Server error",
		}, err
	}

	// Create DynamoDB client
	db := dynamodb.NewFromConfig(cfg)

	// Check if the user exists
	exists, msg := UserExists(req.Email, db, "troggle_user")

	return Response{
		StatusCode: 200,
		Exists:     exists,
		Message:    msg,
	}, nil
}

// main starts the Lambda runtime with our handler
func main() {
	lambda.Start(handler)
}
