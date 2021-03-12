package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	numPasswordsToCreate = 20
)

func TestGraphAPIReliability(t *testing.T) {
	subscriptionID, ok := os.LookupEnv("SUBSCRIPTION_ID")
	if !ok {
		require.True(t, ok, "SUBSCRIPTION_ID must be set")
	}

	tenantID, ok := os.LookupEnv("TENANT_ID")
	if !ok {
		require.True(t, ok, "TENANT_ID must be set")
	}

	clientID, ok := os.LookupEnv("CLIENT_ID")
	if !ok {
		require.True(t, ok, "CLIENT_ID must be set")
	}

	clientSecret, ok := os.LookupEnv("CLIENT_SECRET")
	if !ok {
		require.True(t, ok, "CLIENT_SECRET must be set")
	}

	sutApplicationObjectID, ok := os.LookupEnv("SUT_APPLICATION_OBJECT_ID")
	if !ok {
		require.True(t, ok, "SUT_APPLICATION_OBJECT_ID must be set")
	}

	env, err := azure.EnvironmentFromName("AZUREPUBLICCLOUD")
	require.NoError(t, err, "unable to get env")

	provider, err := newAzureProvider(&clientSettings{
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		Environment:    env,
	})
	require.NoError(t, err, "unable to create provider")

	ctx := context.Background()
	result, err := provider.GetApplication(ctx, sutApplicationObjectID)
	require.NoError(t, err, "unable to list application passwords")
	fmt.Println(fmt.Sprintf("number of credentials: %v", len(result.PasswordCredentials)))

	var parallelRequests int
	parallelRequestsRaw, ok := os.LookupEnv("PARALLEL_REQUESTS")
	if ok {
		i, err := strconv.Atoi(parallelRequestsRaw)
		require.NoError(t, err)
		parallelRequests = i
	} else {
		parallelRequests = numPasswordsToCreate
	}

	var wg sync.WaitGroup

	var keyIds []string
	var mutex sync.Mutex // Note, could use a chan to gather the keyIds and avoid locking
	for i := 0; i < parallelRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			exp := time.Now().Add(24 * time.Hour)
			resp, err := provider.AddApplicationPassword(ctx, sutApplicationObjectID, "", exp)
			require.NoError(t, err, "error adding password")

			mutex.Lock()
			defer mutex.Unlock()

			keyId := to.String(resp.KeyID)
			keyIds = append(keyIds, keyId)

			fmt.Println(fmt.Sprintf("Created password [keyId: %v] for request [ID: %v] at %v", keyId, resp.Header["Request-Id"][0], resp.Header["Date"][0]))
		}()
	}
	wg.Wait()

	// Give time for any propagation delays to resolve before listing the secrets
	time.Sleep(10 * time.Second)

	result, err = provider.GetApplication(ctx, sutApplicationObjectID)
	require.NoError(t, err, "unable to list application passwords")

	var foundKeyIds []string
	for _, cred := range result.PasswordCredentials {
		keyId := to.String(cred.KeyID)
		foundKeyIds = append(foundKeyIds, keyId)
	}

	fmt.Println(fmt.Sprintf("number of credentials: %v", len(foundKeyIds)))
	for _, createdKeyId := range keyIds {
		assert.Contains(t, foundKeyIds, createdKeyId, "key was created but not seemingly not persisted")
	}
}
