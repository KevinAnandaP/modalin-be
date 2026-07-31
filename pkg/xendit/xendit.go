package xendit

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	SecretKey    string
	WebhookToken string
	HTTPClient   *http.Client
	BaseURL      string
}

func NewClient(secretKey, webhookToken string) *Client {
	return &Client{
		SecretKey:    secretKey,
		WebhookToken: webhookToken,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		BaseURL: "https://api.xendit.co",
	}
}

// Request & Response Data Structures for Invoices

type CreateInvoiceReq struct {
	ExternalID      string  `json:"external_id"`
	Amount          float64 `json:"amount"`
	PayerEmail      string  `json:"payer_email,omitempty"`
	Description     string  `json:"description,omitempty"`
	SuccessRedirect string  `json:"success_redirect_url,omitempty"`
	FailureRedirect string  `json:"failure_redirect_url,omitempty"`
	Currency        string  `json:"currency,omitempty"`
}

type InvoiceResp struct {
	ID                 string    `json:"id"`
	ExternalID         string    `json:"external_id"`
	UserID             string    `json:"user_id"`
	Status             string    `json:"status"`
	MerchantName       string    `json:"merchant_name"`
	Amount             float64   `json:"amount"`
	PayerEmail         string    `json:"payer_email"`
	Description        string    `json:"description"`
	InvoiceURL         string    `json:"invoice_url"`
	ExpiryDate         time.Time `json:"expiry_date"`
	PaymentMethod      string    `json:"payment_method"`
	PaymentChannel     string    `json:"payment_channel"`
	PaymentDestination string    `json:"payment_destination"`
}

// Request & Response Data Structures for Disbursements

type CreateDisbursementReq struct {
	ExternalID          string  `json:"external_id"`
	Amount              float64 `json:"amount"`
	BankCode            string  `json:"bank_code"`
	AccountHolderName   string  `json:"account_holder_name"`
	AccountNumber       string  `json:"account_number"`
	Description         string  `json:"description,omitempty"`
	XenditEmailTo       []string `json:"email_to,omitempty"`
}

type DisbursementResp struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	ExternalID        string    `json:"external_id"`
	Amount            float64   `json:"amount"`
	BankCode          string    `json:"bank_code"`
	AccountHolderName string    `json:"account_holder_name"`
	AccountNumber     string    `json:"account_number"`
	Description       string    `json:"description"`
	Status            string    `json:"status"`
	FailureCode       string    `json:"failure_code,omitempty"`
	IsInstant         bool      `json:"is_instant"`
}

// Invoice Webhook Payload struct
type InvoiceWebhookPayload struct {
	ID                 string    `json:"id"`
	ExternalID         string    `json:"external_id"`
	UserID             string    `json:"user_id"`
	Status             string    `json:"status"`
	MerchantName       string    `json:"merchant_name"`
	Amount             float64   `json:"amount"`
	PayerEmail         string    `json:"payer_email"`
	PaymentMethod      string    `json:"payment_method"`
	PaymentChannel     string    `json:"payment_channel"`
	PaidAt             *time.Time `json:"paid_at"`
	PaymentDestination string    `json:"payment_destination"`
}

// Disbursement Webhook Payload struct
type DisbursementWebhookPayload struct {
	ID                string    `json:"id"`
	ExternalID        string    `json:"external_id"`
	UserID            string    `json:"user_id"`
	Amount            float64   `json:"amount"`
	BankCode          string    `json:"bank_code"`
	AccountHolderName string    `json:"account_holder_name"`
	AccountNumber     string    `json:"account_number"`
	Status            string    `json:"status"`
	FailureCode       string    `json:"failure_code,omitempty"`
}

func (c *Client) VerifyWebhookToken(tokenHeader string) bool {
	if c.WebhookToken == "" {
		return true // Sandbox / Dev mode fallback if not set
	}
	return c.WebhookToken == tokenHeader
}

// CreateInvoice calls Xendit POST /v2/invoices API
func (c *Client) CreateInvoice(ctx context.Context, req CreateInvoiceReq) (*InvoiceResp, error) {
	if req.Currency == "" {
		req.Currency = "IDR"
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal invoice req: %w", err)
	}

	url := fmt.Sprintf("%s/v2/invoices", c.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	c.setAuthHeader(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute xendit request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("xendit invoice error [%d]: %s", resp.StatusCode, string(respBody))
	}

	var invoiceResp InvoiceResp
	if err := json.Unmarshal(respBody, &invoiceResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal xendit invoice response: %w", err)
	}

	return &invoiceResp, nil
}

// CreateDisbursement calls Xendit POST /disbursements API
func (c *Client) CreateDisbursement(ctx context.Context, req CreateDisbursementReq) (*DisbursementResp, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal disbursement req: %w", err)
	}

	url := fmt.Sprintf("%s/disbursements", c.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	c.setAuthHeader(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute xendit disbursement request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("xendit disbursement error [%d]: %s", resp.StatusCode, string(respBody))
	}

	var disbResp DisbursementResp
	if err := json.Unmarshal(respBody, &disbResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal xendit disbursement response: %w", err)
	}

	return &disbResp, nil
}

func (c *Client) setAuthHeader(req *http.Request) {
	auth := c.SecretKey + ":"
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", basicAuth)
	req.Header.Set("Content-Type", "application/json")
}
