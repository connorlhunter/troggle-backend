package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events" // API Gateway proxy event definitions
	"github.com/aws/aws-lambda-go/lambda" // Lambda Go runtime
	"github.com/aws/aws-sdk-go-v2/aws"    // aws package
	"github.com/aws/aws-sdk-go-v2/config" // AWS SDK config loader
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Request represents the JSON input
type Request struct {
	Email string `json:"email"` // User email to check
}

// Response represents the JSON output
type Response struct {
	Exists bool `json:"exists"`
}

// UserExists checks if a user with the given email exists in the specified DynamoDB table.
// Returns true if the user exists, false otherwise.
func UserExists(email string, db *dynamodb.Client, tableName string) bool {
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
		return false
	}

	//  A Query returns a slice of items, so check its length
	if len(result.Items) > 0 {
		log.Printf("User found: %s", email)
		return true
	}

	log.Printf("User not found: %s", email)
	return false
}

// handler is the Lambda entry point. It receives an API Gateway event,
// extracts the email from the request body, checks DynamoDB, and returns JSON.
func handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var req Request

	// Parse JSON body from API Gateway request
	err := json.Unmarshal([]byte(event.Body), &req)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid request",
		}, nil
	}

	// Load AWS SDK config (credentials, region, etc.)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("Error loading AWS config: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Server error",
		}, nil
	}

	// Create DynamoDB client
	db := dynamodb.NewFromConfig(cfg)

	// Check if the user exists
	exists := UserExists(req.Email, db, "troggle_user")

	// Marshal response into JSON
	respBody, _ := json.Marshal(Response{Exists: exists})

	// Return API Gateway response
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(respBody),
	}, nil
}

// main starts the Lambda runtime with our handler
func main() {
	lambda.Start(handler)
}
