package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.artefactual.dev/tools/mockutil"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient/enums"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient/ssclientmock"
)

func TestCopyTransfer(t *testing.T) {
	t.Parallel()

	type args struct {
		name     string
		path     string
		ssclient func(rec *ssclientmock.MockClientMockRecorder)
	}

	type want struct {
		path     string
		contents fs.Manifest
		err      string
	}

	type test struct {
		args args
		want want
	}

	commonCalls := func(rec *ssclientmock.MockClientMockRecorder) {
		rec.ReadDefaultLocation(
			mockutil.Context(),
			enums.LocationPurposeTS).
			Return(
				&ssclient.Location{
					URI:     "/api/v2/location/440ec678-ef9f-463c-8725-b6222d44c66d/",
					Purpose: enums.LocationPurposeTS,
				},
				nil,
			).
			Times(1)
		rec.ReadProcessingLocation(mockutil.Context()).
			Return(
				&ssclient.Location{
					URI:     "/api/v2/location/c72d6333-b8a8-45a8-846c-1fb9b57e3629/",
					Purpose: enums.LocationPurposeCP,
				},
				nil,
			).
			Times(1)
		rec.ListLocations(mockutil.Context(), "", enums.LocationPurposeTS).
			Return(
				[]*ssclient.Location{
					{
						URI:     "/api/v2/location/440ec678-ef9f-463c-8725-b6222d44c66d/",
						Purpose: enums.LocationPurposeTS,
					},
				},
				nil,
			).
			Times(1)
	}

	sharedDir := fs.NewDir(t, "ccp-shared",
		fs.WithDir("tmp"),
		fs.WithDir("currentlyProcessing"),
	)

	tests := map[string]test{
		"Transfer1": {
			args: args{
				name: "Transfer1",
				path: "/home/archivematica/transfer1",
				ssclient: func(rec *ssclientmock.MockClientMockRecorder) {
					commonCalls(rec)
					rec.MoveFiles(
						mockutil.Context(),
						&ssclient.Location{
							URI:     "/api/v2/location/440ec678-ef9f-463c-8725-b6222d44c66d/",
							Purpose: enums.LocationPurposeTS,
						},
						&ssclient.Location{
							URI:     "/api/v2/location/c72d6333-b8a8-45a8-846c-1fb9b57e3629/",
							Purpose: enums.LocationPurposeCP,
						},
						[][2]string{
							{
								"home/archivematica/transfer1",
								"/tmp/Transfer1/.",
							},
						},
					).DoAndReturn(func(ctx context.Context, ts, cp *ssclient.Location, files [][2]string) error {
						for _, item := range files {
							os.MkdirAll(sharedDir.Join(item[1]), os.FileMode(0o770))
						}
						return nil
					}).Times(1)
				},
			},
			want: want{
				path:     sharedDir.Join("currentlyProcessing/Transfer1"),
				contents: fs.Expected(t, fs.MatchAnyFileMode),
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ssclient := ssclientmock.NewMockClient(gomock.NewController(t))
			tc.args.ssclient(ssclient.EXPECT())

			path, err := copyTransfer(
				context.Background(),
				ssclient,
				sharedDir.Path(),
				sharedDir.Join("tmp"),
				tc.args.name,
				tc.args.path,
			)

			if tc.want.err != "" {
				assert.Error(t, err, tc.want.err)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, path, tc.want.path)
			assert.Assert(t, fs.Equal(tc.want.path, tc.want.contents))
		})
	}
}

func TestDetermineTransferPaths(t *testing.T) {
	t.Parallel()

	type args struct {
		sharedDir, tmpDir, name, path string
	}

	type want struct {
		destRel, destAbs, src string
	}

	type test struct {
		args args
		want want
	}

	tests := []test{
		{
			args{
				sharedDir: "/var/archivematica/sharedDirectory",
				tmpDir:    "/var/archivematica/sharedDirectory/tmp/tmp.12345",
				name:      "Name1",
				path:      "/var/source/transfer.tar.gz",
			},
			want{
				destRel: "/tmp/tmp.12345",
				destAbs: "/var/archivematica/sharedDirectory/tmp/tmp.12345/transfer.tar.gz",
				src:     "/var/source/transfer.tar.gz",
			},
		},
		{
			args{
				sharedDir: "/var/archivematica/sharedDirectory",
				tmpDir:    "/var/archivematica/sharedDirectory/tmp/tmp.12345",
				name:      "Name2",
				path:      "/var/source/transfer",
			},
			want{
				destRel: "/tmp/tmp.12345/Name2",
				destAbs: "/var/archivematica/sharedDirectory/tmp/tmp.12345/Name2",
				src:     "/var/source/transfer/.",
			},
		},
		{
			args{
				sharedDir: "/var/archivematica/sharedDirectory",
				tmpDir:    "/var/archivematica/sharedDirectory/tmp/tmp.12345",
				name:      "NameWithLocation",
				path:      "cae8fe7a-0ad4-495f-abf5-9d3dbd71ba36:/var/source/transfer.tar.gz",
			},
			want{
				destRel: "/tmp/tmp.12345",
				destAbs: "/var/archivematica/sharedDirectory/tmp/tmp.12345/transfer.tar.gz",
				src:     "cae8fe7a-0ad4-495f-abf5-9d3dbd71ba36:/var/source/transfer.tar.gz",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.args.name, func(t *testing.T) {
			t.Parallel()

			transferRel, filePath, path := determineTransferPaths(
				tc.args.sharedDir, tc.args.tmpDir, tc.args.name, tc.args.path,
			)

			assert.Equal(t, transferRel, tc.want.destRel, "unexpected transferRel")
			assert.Equal(t, filePath, tc.want.destAbs, "unexpected filePath")
			assert.Equal(t, path, tc.want.src, "unexpected path")
		})
	}
}

func TestMoveToInternalSharedDir(t *testing.T) {
	t.Parallel()

	tmpDir := fs.NewDir(t, "ccp",
		fs.WithDir("source",
			fs.WithDir("Images",
				fs.WithFile("MARBLES.TGA", "contents"),
			),
		),
		fs.WithDir("sharedDir",
			fs.WithDir("deposits",
				fs.WithDir("Images"),
				fs.WithDir("Images_1"),
				fs.WithDir("Images_2"),
			),
		),
	)

	dest, err := moveToInternalSharedDir(
		tmpDir.Join("source", "Images"),
		tmpDir.Join("sharedDir", "deposits"),
	)

	assert.NilError(t, err)
	assert.Equal(t, dest, tmpDir.Join("sharedDir", "deposits", filepath.Base(dest)))
	assert.Assert(t, fs.Equal(dest, fs.Expected(t, fs.WithFile("MARBLES.TGA", "contents"), fs.MatchAnyFileMode)))
}
