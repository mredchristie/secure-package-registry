package private_test

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.duti.dev/secure-package-registry/cmd/core-svc/handlers/private"
	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/internal/messages"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockQuerier struct {
	insertPackageFn           func(ctx context.Context, arg coredb.InsertPackageParams) (int32, error)
	listPackagesByEcosystemFn func(ctx context.Context, ecosystem coredb.Ecosystem) ([]coredb.ListPackagesByEcosystemRow, error)
}

func (m *mockQuerier) InsertPackage(ctx context.Context, arg coredb.InsertPackageParams) (int32, error) {
	return m.insertPackageFn(ctx, arg)
}

func (m *mockQuerier) ListPackagesByEcosystem(ctx context.Context, ecosystem coredb.Ecosystem) ([]coredb.ListPackagesByEcosystemRow, error) {
	return m.listPackagesByEcosystemFn(ctx, ecosystem)
}

func (m *mockQuerier) GetPackageVersion(context.Context, coredb.GetPackageVersionParams) (coredb.GetPackageVersionRow, error) {
	panic("not used")
}

func (m *mockQuerier) GetPackageVersionTags(context.Context, coredb.GetPackageVersionTagsParams) ([]coredb.GetPackageVersionTagsRow, error) {
	panic("not used")
}

func (m *mockQuerier) InsertPackageTag(context.Context, coredb.InsertPackageTagParams) error {
	panic("not used")
}

func (m *mockQuerier) InsertPackageVersion(context.Context, coredb.InsertPackageVersionParams) (int32, error) {
	panic("not used")
}

func (m *mockQuerier) InsertTagType(context.Context, coredb.InsertTagTypeParams) (int32, error) {
	panic("not used")
}

func (m *mockQuerier) ListPackageVersions(context.Context, int32) ([]string, error) {
	panic("not used")
}

func (m *mockQuerier) SearchPackages(context.Context, coredb.SearchPackagesParams) ([]coredb.SearchPackagesRow, error) {
	panic("not used")
}

func (m *mockQuerier) UpdatePackageLatestVersion(context.Context, coredb.UpdatePackageLatestVersionParams) error {
	panic("not used")
}

type mockPublisher struct {
	published []*publishedMessage
	err       error
}

type publishedMessage struct {
	topic   string
	payload []byte
}

func (m *mockPublisher) Publish(topic string, msgs ...*message.Message) error {
	if m.err != nil {
		return m.err
	}
	for _, msg := range msgs {
		m.published = append(m.published, &publishedMessage{topic: topic, payload: msg.Payload})
	}
	return nil
}

func (m *mockPublisher) Close() error { return nil }

func newRequest(t *testing.T, body any) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	return httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
}

func decodePublished(t *testing.T, payload []byte) messages.PackageRequest {
	t.Helper()
	var req messages.PackageRequest
	err := gob.NewDecoder(bytes.NewReader(payload)).Decode(&req)
	require.NoError(t, err)
	return req
}

func TestCreatePackage(t *testing.T) {
	t.Parallel()

	t.Run("creates package and publishes request", func(t *testing.T) {
		t.Parallel()
		db := &mockQuerier{
			insertPackageFn: func(_ context.Context, arg coredb.InsertPackageParams) (int32, error) {
				assert.Equal(t, "express", arg.Identifier)
				assert.Equal(t, coredb.EcosystemNpm, arg.Ecosystem)
				assert.False(t, arg.LatestVersion.Valid)
				return 42, nil
			},
		}
		pub := &mockPublisher{}
		handler := private.NewPackageHandler(db, pub)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest(t, map[string]string{
			"identifier": "express",
			"ecosystem":  "npm",
		}))

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp private.CreatePackageResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.Equal(t, int32(42), resp.ID)
		assert.Equal(t, "express", resp.Identifier)
		assert.False(t, resp.AlreadyExists)

		require.Len(t, pub.published, 1)
		assert.Equal(t, "spr.package.requested", pub.published[0].topic)
		msg := decodePublished(t, pub.published[0].payload)
		assert.Equal(t, "npm", msg.Ecosystem)
		assert.Equal(t, "express", msg.Identifier)
	})

	t.Run("returns existing package on duplicate", func(t *testing.T) {
		t.Parallel()
		db := &mockQuerier{
			insertPackageFn: func(context.Context, coredb.InsertPackageParams) (int32, error) {
				return 0, &pgconn.PgError{Code: "23505"}
			},
			listPackagesByEcosystemFn: func(context.Context, coredb.Ecosystem) ([]coredb.ListPackagesByEcosystemRow, error) {
				return []coredb.ListPackagesByEcosystemRow{
					{ID: 7, Identifier: "express", Ecosystem: coredb.EcosystemNpm, LatestVersion: pgtype.Text{String: "4.18.0", Valid: true}},
				}, nil
			},
		}
		pub := &mockPublisher{}
		handler := private.NewPackageHandler(db, pub)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest(t, map[string]string{
			"identifier": "express",
			"ecosystem":  "npm",
		}))

		assert.Equal(t, http.StatusOK, w.Code)

		var resp private.CreatePackageResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.Equal(t, int32(7), resp.ID)
		assert.True(t, resp.AlreadyExists)

		require.Len(t, pub.published, 1)
	})

	t.Run("rejects missing identifier", func(t *testing.T) {
		t.Parallel()
		handler := private.NewPackageHandler(&mockQuerier{}, &mockPublisher{})

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest(t, map[string]string{
			"ecosystem": "npm",
		}))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects missing ecosystem", func(t *testing.T) {
		t.Parallel()
		handler := private.NewPackageHandler(&mockQuerier{}, &mockPublisher{})

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest(t, map[string]string{
			"identifier": "express",
		}))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects invalid ecosystem", func(t *testing.T) {
		t.Parallel()
		handler := private.NewPackageHandler(&mockQuerier{}, &mockPublisher{})

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest(t, map[string]string{
			"identifier": "express",
			"ecosystem":  "rubygems",
		}))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 500 on non-duplicate db error", func(t *testing.T) {
		t.Parallel()
		db := &mockQuerier{
			insertPackageFn: func(context.Context, coredb.InsertPackageParams) (int32, error) {
				return 0, assert.AnError
			},
		}
		handler := private.NewPackageHandler(db, &mockPublisher{})

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest(t, map[string]string{
			"identifier": "express",
			"ecosystem":  "npm",
		}))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("still succeeds when publish fails", func(t *testing.T) {
		t.Parallel()
		db := &mockQuerier{
			insertPackageFn: func(context.Context, coredb.InsertPackageParams) (int32, error) {
				return 1, nil
			},
		}
		pub := &mockPublisher{err: assert.AnError}
		handler := private.NewPackageHandler(db, pub)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, newRequest(t, map[string]string{
			"identifier": "express",
			"ecosystem":  "npm",
		}))

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("accepts all valid ecosystems", func(t *testing.T) {
		t.Parallel()
		for _, eco := range []string{"npm", "go", "cargo", "pypi"} {
			db := &mockQuerier{
				insertPackageFn: func(context.Context, coredb.InsertPackageParams) (int32, error) {
					return 1, nil
				},
			}
			handler := private.NewPackageHandler(db, &mockPublisher{})

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, newRequest(t, map[string]string{
				"identifier": "pkg",
				"ecosystem":  eco,
			}))

			assert.Equal(t, http.StatusCreated, w.Code, "ecosystem %s should be valid", eco)
		}
	})
}
