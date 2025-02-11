//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditTrailsList(t *testing.T) {
	skipIfEnterprise(t)

	userClient := testClient(t)
	ctx := context.Background()

	org, orgCleanup := createOrganization(t, userClient)
	t.Cleanup(orgCleanup)

	auditTrailClient := testAuditTrailClient(t, userClient, org)

	// First let's generate some audit events in this test organization
	_, wkspace1Cleanup := createWorkspace(t, userClient, org)
	t.Cleanup(wkspace1Cleanup)

	_, wkspace2Cleanup := createWorkspace(t, userClient, org)
	t.Cleanup(wkspace2Cleanup)

	t.Run("with no specified timeframe", func(t *testing.T) {
		atl, err := auditTrailClient.AuditTrails.List(ctx, nil)
		require.NoError(t, err)
		require.NotEmpty(t, atl.Items)

		log := atl.Items[0]
		assert.NotEmpty(t, log.ID)
		assert.NotEmpty(t, log.Timestamp)
		assert.NotEmpty(t, log.Type)
		assert.NotEmpty(t, log.Version)
		require.NotNil(t, log.Resource)
		require.NotNil(t, log.Auth)
		require.NotNil(t, log.Request)

		t.Run("with resource deserialized correctly", func(t *testing.T) {
			assert.NotEmpty(t, log.Resource.ID)
			assert.NotEmpty(t, log.Resource.Type)
			assert.NotEmpty(t, log.Resource.Action)

			// we don't test against log.Resource.Meta since we don't know the nature
			// of the audit trail log we're testing against as it can be nil or contain a k-v map
		})

		t.Run("with auth deserialized correctly", func(t *testing.T) {
			assert.NotEmpty(t, log.Auth.AccessorID)
			assert.NotEmpty(t, log.Auth.Description)
			assert.NotEmpty(t, log.Auth.Type)
			assert.NotEmpty(t, log.Auth.OrganizationID)
		})

		t.Run("with request deserialized correctly", func(t *testing.T) {
			assert.NotEmpty(t, log.Request.ID)
		})
	})

	t.Run("using since query param", func(t *testing.T) {
		since := time.Now()

		// Wait some time before creating the event
		// otherwise comparing time values can be flaky
		time.Sleep(1 * time.Second)

		// Let's create an event that is sent to the audit log
		_, wsCleanup := createWorkspace(t, userClient, org)
		t.Cleanup(wsCleanup)

		atl, err := auditTrailClient.AuditTrails.List(ctx, &AuditTrailListOptions{
			Since: since,
			ListOptions: &ListOptions{
				PageNumber: 1,
				PageSize:   20,
			},
		})
		require.NoError(t, err)

		require.Greater(t, len(atl.Items), 0)
		assert.LessOrEqual(t, len(atl.Items), 20)

		for _, log := range atl.Items {
			assert.True(t, log.Timestamp.After(since))
		}
	})
}
