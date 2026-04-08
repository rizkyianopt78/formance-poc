package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net/http"

	formancesdkgo "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
)

// -----------------------------------------------------------------
// Configuration
// Gateway is port-forwarded to localhost:8080 via:
//   kubectl port-forward svc/gateway 8080:8080 -n formance-dev
//
// Ledger API v2 base path: /api/ledger/v2
// -----------------------------------------------------------------
const (
	gatewayURL = "http://localhost:8080"
	ledgerName = "default" // name of the ledger to use
)

func main() {
	// ------------------------------------------------------------------
	// No auth service deployed yet → use a plain HTTP client with no
	// OAuth. The Formance SDK still needs to be initialised with a URL.
	// We override the HTTP client to skip the token fetch entirely.
	// ------------------------------------------------------------------
	client := formancesdkgo.New(
		formancesdkgo.WithServerURL(gatewayURL),
		// Use a no-op http client so the SDK doesn't try to call an
		// OAuth token endpoint (we have no auth service running locally).
		formancesdkgo.WithClient(&http.Client{}),
	)

	ctx := context.Background()

	// ------------------------------------------------------------------
	// 1. Check ledger server info (v2 endpoint: GET /api/ledger/_info)
	// ------------------------------------------------------------------
	fmt.Println("=== Ledger Server Info ===")
	infoRes, err := client.Ledger.V2.GetInfo(ctx)
	if err != nil {
		log.Printf("⚠️  GetInfo failed (ledger may not be reachable yet): %v\n", err)
	} else {
		fmt.Printf("✅ Ledger version: %s\n", infoRes.V2ConfigInfoResponse.Data.Version)
		fmt.Printf("   Storage driver: %s\n", infoRes.V2ConfigInfoResponse.Data.Config.Storage.Driver)
	}

	// ------------------------------------------------------------------
	// 2. Create a ledger (idempotent - safe to call multiple times)
	// POST /api/ledger/v2/{ledger}
	// ------------------------------------------------------------------
	fmt.Printf("\n=== Create Ledger: %q ===\n", ledgerName)
	_, err = client.Ledger.V2.CreateLedger(ctx, operations.V2CreateLedgerRequest{
		Ledger: ledgerName,
		V2CreateLedgerRequest: &shared.V2CreateLedgerRequest{
			Metadata: map[string]string{
				"environment": "dev",
				"owner":       "formance-poc",
			},
		},
	})
	if err != nil {
		// 409 = already exists, that's fine
		log.Printf("ℹ️  CreateLedger: %v\n", err)
	} else {
		fmt.Printf("✅ Ledger %q created\n", ledgerName)
	}

	// ------------------------------------------------------------------
	// 3. Create a transaction: send $1.00 from 'world' to 'alice'
	//    Using the v2 API (POST /api/ledger/v2/{ledger}/transactions)
	// ------------------------------------------------------------------
	fmt.Println("\n=== Create Transaction ===")
	txRes, err := client.Ledger.V2.CreateTransaction(ctx, operations.V2CreateTransactionRequest{
		Ledger: ledgerName,
		V2PostTransaction: shared.V2PostTransaction{
			Postings: []shared.V2Posting{
				{
					Amount:      big.NewInt(100), // 100 cents = $1.00 (USD/2)
					Asset:       "USD/2",
					Source:      "world",
					Destination: "users:alice",
				},
			},
			Metadata: map[string]string{
				"order_id":    "ORD-12345",
				"description": "First v2 transaction via local gateway",
			},
		},
	})
	if err != nil {
		log.Fatal("❌ CreateTransaction failed:", err)
	}
	fmt.Println("✅ Transaction created!")
	fmt.Println("   Transaction ID:", txRes.V2CreateTransactionResponse.Data.ID)

	// ------------------------------------------------------------------
	// 4. Read back alice's balance
	//    GET /api/ledger/v2/{ledger}/accounts/users:alice
	// ------------------------------------------------------------------
	fmt.Println("\n=== Alice's Account ===")
	acctRes, err := client.Ledger.V2.GetAccount(ctx, operations.V2GetAccountRequest{
		Ledger:  ledgerName,
		Address: "users:alice",
	})
	if err != nil {
		log.Printf("⚠️  GetAccount failed: %v\n", err)
	} else {
		account := acctRes.V2AccountResponse.Data
		fmt.Printf("✅ Address: %s\n", account.Address)
		fmt.Printf("   Balances: %v\n", account.Balances)
	}

	// ------------------------------------------------------------------
	// 5. List recent transactions
	//    GET /api/ledger/v2/{ledger}/transactions
	// ------------------------------------------------------------------
	fmt.Println("\n=== Recent Transactions ===")
	listRes, err := client.Ledger.V2.ListTransactions(ctx, operations.V2ListTransactionsRequest{
		Ledger:   ledgerName,
		PageSize: formancesdkgo.Int64(5),
	})
	if err != nil {
		log.Printf("⚠️  ListTransactions failed: %v\n", err)
	} else {
		txs := listRes.V2TransactionsCursorResponse.Cursor.Data
		fmt.Printf("✅ Found %d transaction(s)\n", len(txs))
		for _, tx := range txs {
			fmt.Printf("   [%d] %s  postings=%d  metadata=%v\n",
				tx.ID, tx.Timestamp, len(tx.Postings), tx.Metadata)
		}
	}
}