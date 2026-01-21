#!/bin/bash

# Test Email Sending Script for Kept
# This script tests the SMTP configuration by sending a test email

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== Kept Email Test ===${NC}\n"

# Check if API is running
echo "Checking if Kept API is running..."
if ! curl -s http://localhost:3000/health > /dev/null; then
    echo -e "${RED}Error: Kept API is not running on http://localhost:3000${NC}"
    echo "Please start the backend first with: cd backend && go run main.go"
    exit 1
fi
echo -e "${GREEN}✓ API is running${NC}\n"

# Check if JWT token is provided
if [ -z "$JWT_TOKEN" ]; then
    echo -e "${YELLOW}JWT_TOKEN not set. You need to login first.${NC}"
    echo "To get a token, login with:"
    echo -e "${YELLOW}curl -X POST http://localhost:3000/api/auth/login -H 'Content-Type: application/json' -d '{\"username\":\"your-username\",\"password\":\"your-password\"}'${NC}\n"
    echo "Then set the token:"
    echo -e "${YELLOW}export JWT_TOKEN='your-token-here'${NC}\n"
    exit 1
fi

# Send test email
echo "Sending test email..."
response=$(curl -s -w "\n%{http_code}" -X POST http://localhost:3000/api/email/test \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -H "Content-Type: application/json")

# Split response and status code
status_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$status_code" -eq 200 ]; then
    echo -e "${GREEN}✓ Test email sent successfully!${NC}\n"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
elif [ "$status_code" -eq 429 ]; then
    echo -e "${YELLOW}⚠ Rate limit reached${NC}"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
    echo -e "\n${YELLOW}Email tests are limited to once per 10 minutes.${NC}"
elif [ "$status_code" -eq 503 ]; then
    echo -e "${RED}✗ SMTP not configured${NC}"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
    echo -e "\n${YELLOW}Please configure SMTP environment variables:${NC}"
    echo "SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM"
else
    echo -e "${RED}✗ Failed to send test email (HTTP $status_code)${NC}"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
fi
